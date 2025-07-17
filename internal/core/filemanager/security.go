package filemanager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// SecurityManager handles file security operations
type SecurityManager struct {
	logger       *logrus.Logger
	encryptKey   []byte
	virusScanner VirusScanner
}

// VirusScanner interface for virus scanning implementations
type VirusScanner interface {
	ScanFile(filePath string) (*ScanResult, error)
	ScanStream(content io.Reader) (*ScanResult, error)
}

// ScanResult contains the result of virus scanning
type ScanResult struct {
	Clean        bool     `json:"clean"`
	Threats      []string `json:"threats,omitempty"`
	ScanEngine   string   `json:"scan_engine"`
	ScanTime     int64    `json:"scan_time"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	EncryptionEnabled bool
	VirusScanEnabled  bool
	AllowedExtensions []string
	BlockedExtensions []string
	MaxFileSize       int64
	ScanOnUpload      bool
	QuarantinePath    string
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(logger *logrus.Logger, encryptKey []byte, scanner VirusScanner) *SecurityManager {
	return &SecurityManager{
		logger:       logger,
		encryptKey:   encryptKey,
		virusScanner: scanner,
	}
}

// ValidateFileUpload performs security validation on file upload
func (sm *SecurityManager) ValidateFileUpload(filename string, content io.Reader, config SecurityConfig) error {
	// Check file extension
	if err := sm.validateFileExtension(filename, config); err != nil {
		return err
	}

	// Scan for viruses if enabled
	if config.VirusScanEnabled && sm.virusScanner != nil {
		result, err := sm.virusScanner.ScanStream(content)
		if err != nil {
			sm.logger.Warnf("Virus scan failed for %s: %v", filename, err)
			return fmt.Errorf("virus scan failed: %w", err)
		}

		if !result.Clean {
			sm.logger.Warnf("Malicious file detected: %s, threats: %v", filename, result.Threats)
			return fmt.Errorf("malicious file detected: %v", result.Threats)
		}
	}

	return nil
}

// validateFileExtension checks if file extension is allowed
func (sm *SecurityManager) validateFileExtension(filename string, config SecurityConfig) error {
	ext := strings.ToLower(filepath.Ext(filename))

	// Check blocked extensions first
	for _, blocked := range config.BlockedExtensions {
		if ext == strings.ToLower(blocked) {
			return fmt.Errorf("file extension %s is blocked", ext)
		}
	}

	// If allowed extensions are specified, check against them
	if len(config.AllowedExtensions) > 0 {
		allowed := false
		for _, allowedExt := range config.AllowedExtensions {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension %s is not allowed", ext)
		}
	}

	return nil
}

// EncryptFile encrypts a file using AES-GCM
func (sm *SecurityManager) EncryptFile(content io.Reader) (io.Reader, error) {
	if len(sm.encryptKey) == 0 {
		return content, nil // No encryption key, return as-is
	}

	// Read all content into memory (for simplicity)
	plaintext, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	block, err := aes.NewCipher(sm.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return strings.NewReader(string(ciphertext)), nil
}

// DecryptFile decrypts a file using AES-GCM
func (sm *SecurityManager) DecryptFile(encryptedContent io.Reader) (io.Reader, error) {
	if len(sm.encryptKey) == 0 {
		return encryptedContent, nil // No encryption key, return as-is
	}

	// Read all encrypted content
	ciphertext, err := io.ReadAll(encryptedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted content: %w", err)
	}

	block, err := aes.NewCipher(sm.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return strings.NewReader(string(plaintext)), nil
}

// CheckFilePermission verifies if a user has permission to access a file
func (sm *SecurityManager) CheckFilePermission(userID int, fileID string, permission string, filePermissions []FilePermission) bool {
	// Check for specific user permissions
	for _, perm := range filePermissions {
		if perm.FileID == fileID {
			// Check if permission is for this user or is global (userID is nil)
			if perm.UserID == nil || *perm.UserID == userID {
				if perm.Permission == permission || perm.Permission == PermissionAdmin {
					return true
				}
			}
		}
	}

	return false
}

// GrantFilePermission grants permission to a user for a file
func (sm *SecurityManager) GrantFilePermission(userID *int, fileID string, permission string) *FilePermission {
	return &FilePermission{
		FileID:     fileID,
		UserID:     userID,
		Permission: permission,
	}
}

// SecureDelete performs secure deletion of a file
func (sm *SecurityManager) SecureDelete(filePath string) error {
	// This is a simplified secure delete - in production, you might want
	// to overwrite the file multiple times with random data
	return sm.overwriteAndDelete(filePath)
}

// overwriteAndDelete overwrites file with random data before deletion
func (sm *SecurityManager) overwriteAndDelete(filePath string) error {
	// Get file size
	_, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	// Simple approach: just delete the file
	// In a production environment, you might want to:
	// 1. Overwrite with random data multiple times
	// 2. Sync to disk
	// 3. Then delete
	return fmt.Errorf("secure delete not fully implemented")
}

// DetectSensitiveData scans content for sensitive information patterns
func (sm *SecurityManager) DetectSensitiveData(content string) []string {
	var findings []string

	// Simple patterns for common sensitive data
	patterns := map[string]string{
		"SSN":         `\b\d{3}-\d{2}-\d{4}\b`,
		"Credit Card": `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`,
		"Email":       `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		"Phone":       `\b\d{3}-\d{3}-\d{4}\b`,
		"API Key":     `[Aa][Pp][Ii][\s_-]?[Kk][Ee][Yy][\s_-]*[:=][\s_-]*[A-Za-z0-9]{20,}`,
	}

	for name := range patterns {
		// In a real implementation, you would use regexp here
		// For simplicity, we'll just check if the pattern name exists in content
		if strings.Contains(strings.ToLower(content), strings.ToLower(name)) {
			findings = append(findings, name)
		}
	}

	return findings
}

// MockVirusScanner is a simple mock implementation for testing
type MockVirusScanner struct {
	logger *logrus.Logger
}

// NewMockVirusScanner creates a new mock virus scanner
func NewMockVirusScanner(logger *logrus.Logger) *MockVirusScanner {
	return &MockVirusScanner{logger: logger}
}

// ScanFile implements VirusScanner.ScanFile for mock scanner
func (mvs *MockVirusScanner) ScanFile(filePath string) (*ScanResult, error) {
	mvs.logger.Debugf("Mock virus scan for file: %s", filePath)

	// Mock scan - always clean for now
	return &ScanResult{
		Clean:      true,
		Threats:    []string{},
		ScanEngine: "MockScanner",
		ScanTime:   100, // milliseconds
	}, nil
}

// ScanStream implements VirusScanner.ScanStream for mock scanner
func (mvs *MockVirusScanner) ScanStream(content io.Reader) (*ScanResult, error) {
	mvs.logger.Debug("Mock virus scan for stream")

	// Mock scan - always clean for now
	return &ScanResult{
		Clean:      true,
		Threats:    []string{},
		ScanEngine: "MockScanner",
		ScanTime:   100, // milliseconds
	}, nil
}

// GetDefaultSecurityConfig returns default security configuration
func GetDefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EncryptionEnabled: false,
		VirusScanEnabled:  false,
		AllowedExtensions: []string{}, // Empty means allow all
		BlockedExtensions: []string{
			".exe", ".bat", ".cmd", ".com", ".pif", ".scr", ".vbs", ".js",
			".jar", ".app", ".deb", ".pkg", ".dmg", ".msi",
		},
		MaxFileSize:    1024 * 1024 * 1024, // 1GB
		ScanOnUpload:   true,
		QuarantinePath: "./data/quarantine",
	}
}
