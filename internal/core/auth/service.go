package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	SessionTimeout    int    // Session timeout in seconds
	MaxFailedAttempts int    // Maximum failed attempts before lockout
	LockoutDuration   int    // Lockout duration in seconds
	JWTSecret         string // JWT secret for token signing
}

// Service provides enhanced authentication functionality
type Service struct {
	repo   repositories.AuthRepository
	config AuthConfig
	logger *logrus.Logger

	// Rate limiting
	failedAttempts map[string][]*time.Time
	lockouts       map[string]*time.Time
	mu             sync.RWMutex

	// Session cache
	sessionCache map[string]*models.Session
	cacheMu      sync.RWMutex

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewService creates a new authentication service
func NewService(repo repositories.AuthRepository, config AuthConfig, logger *logrus.Logger) *Service {
	if config.SessionTimeout == 0 {
		config.SessionTimeout = 300 // 5 minutes default
	}
	if config.MaxFailedAttempts == 0 {
		config.MaxFailedAttempts = 3
	}
	if config.LockoutDuration == 0 {
		config.LockoutDuration = 300 // 5 minutes default
	}

	service := &Service{
		repo:           repo,
		config:         config,
		logger:         logger,
		failedAttempts: make(map[string][]*time.Time),
		lockouts:       make(map[string]*time.Time),
		sessionCache:   make(map[string]*models.Session),
		cleanupTicker:  time.NewTicker(5 * time.Minute), // Cleanup every 5 minutes
		stopCleanup:    make(chan bool),
	}

	// Start background cleanup
	go service.backgroundCleanup()

	return service
}

// Initialize initializes the authentication service
func (s *Service) Initialize(ctx context.Context) error {
	// Ensure auth settings exist with defaults
	_, err := s.repo.GetSettings(ctx)
	if err != nil {
		// Create default settings if they don't exist
		defaultSettings := &models.AuthSetting{
			ID:                1,
			SessionTimeout:    s.config.SessionTimeout,
			MaxFailedAttempts: s.config.MaxFailedAttempts,
			LockoutDuration:   s.config.LockoutDuration,
		}

		if err := s.repo.SetSettings(ctx, defaultSettings); err != nil {
			return fmt.Errorf("failed to create default auth settings: %w", err)
		}

		s.logger.Info("Created default authentication settings")
	}

	// Clean up expired sessions and failed attempts
	if err := s.repo.DeleteExpiredSessions(ctx); err != nil {
		s.logger.WithError(err).Warn("Failed to cleanup expired sessions on startup")
	}

	before := time.Now().Add(-24 * time.Hour).Unix()
	if err := s.repo.CleanupFailedAttempts(ctx, before); err != nil {
		s.logger.WithError(err).Warn("Failed to cleanup old failed attempts on startup")
	}

	s.logger.Info("Authentication service initialized successfully")
	return nil
}

// HasPin checks if a PIN is currently set
func (s *Service) HasPin(ctx context.Context) (bool, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get auth settings: %w", err)
	}

	return settings.PinCode.Valid && settings.PinCode.String != "", nil
}

// SetPin sets a new PIN (only if no PIN is currently set)
func (s *Service) SetPin(ctx context.Context, pin, clientID string) (*SessionResponse, error) {
	if !s.isValidPin(pin) {
		s.recordFailedAttempt(clientID, "pin")
		return nil, fmt.Errorf("invalid PIN format. PIN must be exactly 4 or 6 digits")
	}

	if err := s.checkRateLimit(clientID); err != nil {
		return nil, err
	}

	// Check if PIN is already set
	hasPin, err := s.HasPin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check PIN status: %w", err)
	}

	if hasPin {
		s.recordFailedAttempt(clientID, "pin")
		return nil, fmt.Errorf("PIN is already set. Use change PIN endpoint")
	}

	// Set the new PIN
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth settings: %w", err)
	}

	settings.PinCode = sql.NullString{String: pin, Valid: true}
	if err := s.repo.SetSettings(ctx, settings); err != nil {
		s.recordFailedAttempt(clientID, "pin")
		return nil, fmt.Errorf("failed to save PIN: %w", err)
	}

	s.clearFailedAttempts(clientID)
	s.logger.WithField("client_id", clientID).Info("PIN set successfully")

	// Generate session token for immediate login
	return s.generateSession(ctx, pin)
}

// VerifyPin verifies a PIN and returns session information if valid
func (s *Service) VerifyPin(ctx context.Context, pin, clientID string) (*SessionResponse, error) {
	if err := s.checkRateLimit(clientID); err != nil {
		return nil, err
	}

	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth settings: %w", err)
	}

	if !settings.PinCode.Valid || settings.PinCode.String == "" {
		s.recordFailedAttempt(clientID, "pin")
		return nil, fmt.Errorf("no PIN is set")
	}

	if settings.PinCode.String != pin {
		s.recordFailedAttempt(clientID, "pin")
		return nil, fmt.Errorf("invalid PIN")
	}

	s.clearFailedAttempts(clientID)
	s.logger.WithField("client_id", clientID).Info("PIN verified successfully")

	return s.generateSession(ctx, pin)
}

// ChangePin changes the current PIN
func (s *Service) ChangePin(ctx context.Context, currentPin, newPin, clientID string) error {
	if !s.isValidPin(newPin) {
		s.recordFailedAttempt(clientID, "pin")
		return fmt.Errorf("invalid new PIN format. PIN must be exactly 4 or 6 digits")
	}

	if err := s.checkRateLimit(clientID); err != nil {
		return err
	}

	// Verify current PIN first
	_, err := s.VerifyPin(ctx, currentPin, clientID)
	if err != nil {
		return fmt.Errorf("current PIN is incorrect: %w", err)
	}

	// Set new PIN
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth settings: %w", err)
	}

	settings.PinCode = sql.NullString{String: newPin, Valid: true}
	if err := s.repo.SetSettings(ctx, settings); err != nil {
		s.recordFailedAttempt(clientID, "pin")
		return fmt.Errorf("failed to save new PIN: %w", err)
	}

	s.clearFailedAttempts(clientID)
	s.logger.WithField("client_id", clientID).Info("PIN changed successfully")

	return nil
}

// DisablePin disables PIN authentication
func (s *Service) DisablePin(ctx context.Context, currentPin, clientID string) error {
	if err := s.checkRateLimit(clientID); err != nil {
		return err
	}

	// Verify current PIN first
	_, err := s.VerifyPin(ctx, currentPin, clientID)
	if err != nil {
		return fmt.Errorf("current PIN is incorrect: %w", err)
	}

	// Clear the PIN
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth settings: %w", err)
	}

	settings.PinCode = sql.NullString{Valid: false}
	if err := s.repo.SetSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to clear PIN: %w", err)
	}

	// Invalidate all sessions for security
	if err := s.repo.DeleteExpiredSessions(ctx); err != nil {
		s.logger.WithError(err).Warn("Failed to cleanup sessions after PIN disable")
	}

	s.clearFailedAttempts(clientID)
	s.sessionCache = make(map[string]*models.Session) // Clear cache
	s.logger.WithField("client_id", clientID).Info("PIN disabled and all sessions cleared")

	return nil
}

// ValidateSession validates a session token
func (s *Service) ValidateSession(ctx context.Context, token string) (*models.Session, error) {
	// Check cache first
	s.cacheMu.RLock()
	if session, exists := s.sessionCache[token]; exists {
		if session.ExpiresAt.After(time.Now()) {
			s.cacheMu.RUnlock()
			return session, nil
		}
		// Remove expired session from cache
		delete(s.sessionCache, token)
	}
	s.cacheMu.RUnlock()

	// Get from database
	session, err := s.repo.GetSession(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired session")
	}

	// Add to cache
	s.cacheMu.Lock()
	s.sessionCache[token] = session
	s.cacheMu.Unlock()

	return session, nil
}

// InvalidateSession invalidates a specific session
func (s *Service) InvalidateSession(ctx context.Context, token string) error {
	if err := s.repo.DeleteSession(ctx, token); err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}

	// Remove from cache
	s.cacheMu.Lock()
	delete(s.sessionCache, token)
	s.cacheMu.Unlock()

	return nil
}

// GetPinStatus returns information about PIN configuration
func (s *Service) GetPinStatus(ctx context.Context) (*PinStatusResponse, error) {
	hasPin, err := s.HasPin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check PIN status: %w", err)
	}

	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth settings: %w", err)
	}

	response := &PinStatusResponse{
		PinSet:         hasPin,
		SetupComplete:  hasPin,
		SessionTimeout: settings.SessionTimeout,
	}

	if hasPin {
		// Determine PIN length without exposing the actual PIN
		if len(settings.PinCode.String) == 4 {
			response.PinLength = 4
		} else if len(settings.PinCode.String) == 6 {
			response.PinLength = 6
		}
	}

	return response, nil
}

// SessionResponse represents a session creation response
type SessionResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PinStatusResponse represents PIN status information
type PinStatusResponse struct {
	PinSet         bool `json:"pin_set"`
	SetupComplete  bool `json:"setup_complete"`
	PinLength      int  `json:"pin_length,omitempty"`
	SessionTimeout int  `json:"session_timeout"`
}

// Private helper methods

func (s *Service) generateSession(ctx context.Context, pin string) (*SessionResponse, error) {
	// Generate secure token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secure token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Create session
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(time.Duration(s.config.SessionTimeout) * time.Second)

	session := &models.Session{
		ID:        sessionID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Add to cache
	s.cacheMu.Lock()
	s.sessionCache[token] = session
	s.cacheMu.Unlock()

	return &SessionResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) isValidPin(pin string) bool {
	// Allow only 4 or 6 digit PINs
	match4, _ := regexp.MatchString(`^\d{4}$`, pin)
	match6, _ := regexp.MatchString(`^\d{6}$`, pin)
	return match4 || match6
}

func (s *Service) checkRateLimit(clientID string) error {
	s.mu.RLock()

	// Check if client is locked out
	if lockoutTime, exists := s.lockouts[clientID]; exists {
		if time.Now().Before(*lockoutTime) {
			s.mu.RUnlock()
			return fmt.Errorf("account locked due to too many failed attempts. Try again later")
		}
		// Lockout expired, remove it
		s.mu.RUnlock()
		s.mu.Lock()
		delete(s.lockouts, clientID)
		s.mu.Unlock()
		return nil
	}

	// Check failed attempts count
	attempts, exists := s.failedAttempts[clientID]
	if exists && len(attempts) >= s.config.MaxFailedAttempts {
		s.mu.RUnlock()
		return fmt.Errorf("too many failed attempts. Account locked temporarily")
	}

	s.mu.RUnlock()
	return nil
}

func (s *Service) recordFailedAttempt(clientID, attemptType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Add failed attempt
	attempts := s.failedAttempts[clientID]
	attempts = append(attempts, &now)

	// Keep only recent attempts (last hour)
	cutoff := now.Add(-time.Hour)
	var recentAttempts []*time.Time
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			recentAttempts = append(recentAttempts, attempt)
		}
	}

	s.failedAttempts[clientID] = recentAttempts

	// Check if lockout is needed
	if len(recentAttempts) >= s.config.MaxFailedAttempts {
		lockoutUntil := now.Add(time.Duration(s.config.LockoutDuration) * time.Second)
		s.lockouts[clientID] = &lockoutUntil
		s.logger.WithFields(logrus.Fields{
			"client_id":     clientID,
			"attempt_type":  attemptType,
			"lockout_until": lockoutUntil,
		}).Warn("Client locked out due to failed attempts")
	}

	// Store in database for persistence
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		attempt := &models.FailedAuthAttempt{
			ClientID:    clientID,
			IPAddress:   "", // Could be enhanced to include IP
			AttemptType: attemptType,
		}

		if err := s.repo.RecordFailedAttempt(ctx, attempt); err != nil {
			s.logger.WithError(err).Error("Failed to record failed attempt in database")
		}
	}()
}

func (s *Service) clearFailedAttempts(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.failedAttempts, clientID)
	delete(s.lockouts, clientID)
}

func (s *Service) backgroundCleanup() {
	for {
		select {
		case <-s.cleanupTicker.C:
			s.performCleanup()
		case <-s.stopCleanup:
			s.cleanupTicker.Stop()
			return
		}
	}
}

func (s *Service) performCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Cleanup expired sessions from database and cache
	if err := s.repo.DeleteExpiredSessions(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to cleanup expired sessions")
	}

	// Cleanup session cache
	s.cacheMu.Lock()
	now := time.Now()
	for token, session := range s.sessionCache {
		if session.ExpiresAt.Before(now) {
			delete(s.sessionCache, token)
		}
	}
	s.cacheMu.Unlock()

	// Cleanup old failed attempts
	before := time.Now().Add(-24 * time.Hour).Unix()
	if err := s.repo.CleanupFailedAttempts(ctx, before); err != nil {
		s.logger.WithError(err).Error("Failed to cleanup old failed attempts")
	}

	// Cleanup in-memory failed attempts and lockouts
	s.mu.Lock()
	cutoff := time.Now().Add(-time.Hour)
	for clientID, attempts := range s.failedAttempts {
		var recentAttempts []*time.Time
		for _, attempt := range attempts {
			if attempt.After(cutoff) {
				recentAttempts = append(recentAttempts, attempt)
			}
		}
		if len(recentAttempts) == 0 {
			delete(s.failedAttempts, clientID)
		} else {
			s.failedAttempts[clientID] = recentAttempts
		}
	}

	// Cleanup expired lockouts
	now = time.Now()
	for clientID, lockoutTime := range s.lockouts {
		if now.After(*lockoutTime) {
			delete(s.lockouts, clientID)
		}
	}
	s.mu.Unlock()
}

// Shutdown gracefully stops the service
func (s *Service) Shutdown() {
	close(s.stopCleanup)
}
