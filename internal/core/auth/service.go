package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication business logic
type Service struct {
	userRepo    repositories.UserRepository
	jwtSecret   string
	tokenExpiry int
	logger      *logrus.Logger
}

// NewService creates a new authentication service
func NewService(userRepo repositories.UserRepository, jwtSecret string, tokenExpiry int, logger *logrus.Logger) *Service {
	return &Service{
		userRepo:    userRepo,
		jwtSecret:   jwtSecret,
		tokenExpiry: tokenExpiry,
		logger:      logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *UserInfo `json:"user"`
}

// UserInfo represents user information for responses
type UserInfo struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Register creates a new user account
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*UserInfo, error) {
	// Check if username already exists
	existing, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Validate password strength
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters long")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash password")
		return nil, fmt.Errorf("failed to process password")
	}

	// Create user
	user := &models.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to create user: %s", req.Username)
		return nil, fmt.Errorf("failed to create user")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User registered successfully")

	return &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"username": req.Username,
		}).Warn("Login attempt with non-existent username")
		return nil, fmt.Errorf("invalid username or password")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"user_id":  user.ID,
			"username": user.Username,
		}).Warn("Login attempt with incorrect password")
		return nil, fmt.Errorf("invalid username or password")
	}

	// Generate JWT token
	expiresAt := time.Now().Add(time.Duration(s.tokenExpiry) * time.Second)
	claims := &TokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "pma-backend-go",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.logger.WithError(err).Error("Failed to sign JWT token")
		return nil, fmt.Errorf("failed to generate token")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User logged in successfully")

	return &LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: &UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

// GetUserByID retrieves user information by ID
func (s *Service) GetUserByID(ctx context.Context, userID int) (*UserInfo, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

// UpdatePassword updates a user's password
func (s *Service) UpdatePassword(ctx context.Context, userID int, oldPassword, newPassword string) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword))
	if err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Validate new password
	if len(newPassword) < 6 {
		return fmt.Errorf("new password must be at least 6 characters long")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash new password")
		return fmt.Errorf("failed to process new password")
	}

	// Update password
	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	err = s.userRepo.Update(ctx, user)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to update password for user: %d", userID)
		return fmt.Errorf("failed to update password")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"username": user.Username,
	}).Info("User password updated successfully")

	return nil
}

// ValidateToken validates a JWT token and returns user information
func (s *Service) ValidateToken(tokenString string) (*UserInfo, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return &UserInfo{
			ID:       claims.UserID,
			Username: claims.Username,
		}, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// GetAllUsers retrieves all users (admin functionality)
func (s *Service) GetAllUsers(ctx context.Context) ([]*UserInfo, error) {
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get all users")
		return nil, fmt.Errorf("failed to retrieve users")
	}

	userInfos := make([]*UserInfo, len(users))
	for i, user := range users {
		userInfos[i] = &UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			CreatedAt: user.CreatedAt,
		}
	}

	return userInfos, nil
}

// DeleteUser deletes a user account (admin functionality)
func (s *Service) DeleteUser(ctx context.Context, userID int) error {
	err := s.userRepo.Delete(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to delete user: %d", userID)
		return fmt.Errorf("failed to delete user")
	}

	s.logger.WithField("user_id", userID).Info("User deleted successfully")
	return nil
}
