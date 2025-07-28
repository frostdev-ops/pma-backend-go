package filemanager

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// BasicVirusScanner is a simple implementation for basic security checks
type BasicVirusScanner struct {
	logger *logrus.Logger
}

// NewBasicVirusScanner creates a new basic virus scanner
func NewBasicVirusScanner(logger *logrus.Logger) *BasicVirusScanner {
	return &BasicVirusScanner{
		logger: logger,
	}
}

// ScanFile scans a file for potential threats using basic heuristics
func (bvs *BasicVirusScanner) ScanFile(filePath string) (*ScanResult, error) {
	startTime := time.Now()

	file, err := os.Open(filePath)
	if err != nil {
		return &ScanResult{
			Clean:        false,
			ScanEngine:   "BasicScanner",
			ScanTime:     time.Since(startTime).Milliseconds(),
			ErrorMessage: fmt.Sprintf("Failed to open file: %v", err),
		}, err
	}
	defer file.Close()

	return bvs.ScanStream(file)
}

// ScanStream scans a stream for potential threats using basic heuristics
func (bvs *BasicVirusScanner) ScanStream(content io.Reader) (*ScanResult, error) {
	startTime := time.Now()
	threats := []string{}

	// Read content for analysis
	scanner := bufio.NewScanner(content)
	lineNumber := 0

	// Basic heuristic checks
	for scanner.Scan() {
		lineNumber++
		line := strings.ToLower(scanner.Text())

		// Check for suspicious patterns
		if bvs.containsSuspiciousPatterns(line) {
			threats = append(threats, fmt.Sprintf("Suspicious pattern detected at line %d", lineNumber))
		}

		// Limit scanning to first 1000 lines for performance
		if lineNumber >= 1000 {
			break
		}
	}

	result := &ScanResult{
		Clean:      len(threats) == 0,
		Threats:    threats,
		ScanEngine: "BasicScanner",
		ScanTime:   time.Since(startTime).Milliseconds(),
	}

	bvs.logger.WithFields(logrus.Fields{
		"clean":     result.Clean,
		"threats":   len(threats),
		"scan_time": result.ScanTime,
	}).Debug("Basic virus scan completed")

	return result, nil
}

// containsSuspiciousPatterns checks for basic suspicious patterns
func (bvs *BasicVirusScanner) containsSuspiciousPatterns(content string) bool {
	suspiciousPatterns := []string{
		"<script",
		"javascript:",
		"vbscript:",
		"data:text/html",
		"eval(",
		"system(",
		"exec(",
		"shell_exec(",
		"passthru(",
		"<?php",
		"<%",
		"<%=",
		"<%@",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}

	return false
}

// ClamAVScanner implements virus scanning using ClamAV
type ClamAVScanner struct {
	logger      *logrus.Logger
	clamScanCmd string
	enabled     bool
}

// NewClamAVScanner creates a new ClamAV scanner
func NewClamAVScanner(logger *logrus.Logger) *ClamAVScanner {
	scanner := &ClamAVScanner{
		logger:      logger,
		clamScanCmd: "clamscan",
		enabled:     false,
	}

	// Check if ClamAV is available
	if err := scanner.checkClamAVAvailability(); err != nil {
		logger.WithError(err).Warn("ClamAV not available, scanner disabled")
	} else {
		scanner.enabled = true
		logger.Info("ClamAV scanner initialized successfully")
	}

	return scanner
}

// checkClamAVAvailability checks if ClamAV is installed and available
func (cavs *ClamAVScanner) checkClamAVAvailability() error {
	cmd := exec.Command("clamscan", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("clamscan not found: %w", err)
	}

	cavs.logger.WithField("version", string(output)).Info("ClamAV detected")
	return nil
}

// ScanFile scans a file using ClamAV
func (cavs *ClamAVScanner) ScanFile(filePath string) (*ScanResult, error) {
	startTime := time.Now()

	if !cavs.enabled {
		return &ScanResult{
			Clean:        true,
			ScanEngine:   "ClamAV",
			ScanTime:     time.Since(startTime).Milliseconds(),
			ErrorMessage: "ClamAV not available",
		}, nil
	}

	// Run clamscan
	cmd := exec.Command("clamscan", "--no-summary", "--infected", filePath)
	output, err := cmd.Output()

	result := &ScanResult{
		ScanEngine: "ClamAV",
		ScanTime:   time.Since(startTime).Milliseconds(),
	}

	if err != nil {
		// ClamAV returns exit code 1 when virus found, 0 when clean
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				// Virus found
				result.Clean = false
				result.Threats = cavs.parseThreats(string(output))
			} else {
				// Other error
				result.Clean = false
				result.ErrorMessage = fmt.Sprintf("ClamAV scan error: %v", err)
			}
		} else {
			result.Clean = false
			result.ErrorMessage = fmt.Sprintf("Failed to run ClamAV: %v", err)
		}
	} else {
		// Clean file
		result.Clean = true
	}

	cavs.logger.WithFields(logrus.Fields{
		"file":      filePath,
		"clean":     result.Clean,
		"threats":   len(result.Threats),
		"scan_time": result.ScanTime,
	}).Info("ClamAV scan completed")

	return result, nil
}

// ScanStream scans a stream using ClamAV by writing to temporary file
func (cavs *ClamAVScanner) ScanStream(content io.Reader) (*ScanResult, error) {
	if !cavs.enabled {
		return &ScanResult{
			Clean:        true,
			ScanEngine:   "ClamAV",
			ErrorMessage: "ClamAV not available",
		}, nil
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "pma_scan_*.tmp")
	if err != nil {
		return &ScanResult{
			Clean:        false,
			ScanEngine:   "ClamAV",
			ErrorMessage: fmt.Sprintf("Failed to create temp file: %v", err),
		}, err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy content to temp file
	if _, err := io.Copy(tempFile, content); err != nil {
		return &ScanResult{
			Clean:        false,
			ScanEngine:   "ClamAV",
			ErrorMessage: fmt.Sprintf("Failed to write temp file: %v", err),
		}, err
	}

	// Scan the temporary file
	return cavs.ScanFile(tempFile.Name())
}

// parseThreats extracts threat names from ClamAV output
func (cavs *ClamAVScanner) parseThreats(output string) []string {
	var threats []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.Contains(line, "FOUND") {
			// Extract threat name from line like: "/path/to/file: Threat.Name FOUND"
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				threatPart := strings.TrimSpace(parts[1])
				threatName := strings.Replace(threatPart, " FOUND", "", 1)
				threats = append(threats, strings.TrimSpace(threatName))
			}
		}
	}

	return threats
}

// CompositeVirusScanner combines multiple scanners for better coverage
type CompositeVirusScanner struct {
	scanners []VirusScanner
	logger   *logrus.Logger
}

// NewCompositeVirusScanner creates a new composite scanner
func NewCompositeVirusScanner(logger *logrus.Logger, scanners ...VirusScanner) *CompositeVirusScanner {
	return &CompositeVirusScanner{
		scanners: scanners,
		logger:   logger,
	}
}

// ScanFile scans using all available scanners
func (cvs *CompositeVirusScanner) ScanFile(filePath string) (*ScanResult, error) {
	startTime := time.Now()
	var allThreats []string
	var engines []string
	var errors []string

	for _, scanner := range cvs.scanners {
		result, err := scanner.ScanFile(filePath)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		engines = append(engines, result.ScanEngine)
		if !result.Clean {
			allThreats = append(allThreats, result.Threats...)
		}
	}

	result := &ScanResult{
		Clean:      len(allThreats) == 0,
		Threats:    allThreats,
		ScanEngine: strings.Join(engines, ", "),
		ScanTime:   time.Since(startTime).Milliseconds(),
	}

	if len(errors) > 0 {
		result.ErrorMessage = strings.Join(errors, "; ")
	}

	return result, nil
}

// ScanStream scans using all available scanners
func (cvs *CompositeVirusScanner) ScanStream(content io.Reader) (*ScanResult, error) {
	// Read content into buffer so we can scan it multiple times
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, content); err != nil {
		return &ScanResult{
			Clean:        false,
			ScanEngine:   "Composite",
			ErrorMessage: fmt.Sprintf("Failed to read content: %v", err),
		}, err
	}

	startTime := time.Now()
	var allThreats []string
	var engines []string
	var errors []string

	for _, scanner := range cvs.scanners {
		// Create a new reader for each scanner
		reader := bytes.NewReader(buf.Bytes())
		result, err := scanner.ScanStream(reader)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		engines = append(engines, result.ScanEngine)
		if !result.Clean {
			allThreats = append(allThreats, result.Threats...)
		}
	}

	result := &ScanResult{
		Clean:      len(allThreats) == 0,
		Threats:    allThreats,
		ScanEngine: strings.Join(engines, ", "),
		ScanTime:   time.Since(startTime).Milliseconds(),
	}

	if len(errors) > 0 {
		result.ErrorMessage = strings.Join(errors, "; ")
	}

	return result, nil
}
