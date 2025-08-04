package providers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/sirupsen/logrus"
)

// Initialize starts the llama.cpp server if auto_start is enabled
func (p *LlamaCppProvider) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if already running
	if err := p.healthCheck(ctx); err == nil {
		p.isRunning = true
		p.logger.Info("llama.cpp server already running")
		return nil
	}

	// Start server if auto_start is enabled
	if p.llamaCppConfig.AutoStart {
		if err := p.startServer(ctx); err != nil {
			return fmt.Errorf("failed to start llama.cpp server: %w", err)
		}
	}

	// Wait for server to be ready
	return p.waitForReady(ctx, 300*time.Second) // Increased to 5 minutes for F16 model loading
}

// startServer starts the llama.cpp server process
func (p *LlamaCppProvider) startServer(ctx context.Context) error {
	// Find llama.cpp binary
	llamaCppPath, err := p.findLlamaCppBinary()
	if err != nil {
		return fmt.Errorf("llama.cpp binary not found: %w", err)
	}

	// Find model file
	modelPath, err := p.findModelFile()
	if err != nil {
		return fmt.Errorf("model file not found: %w", err)
	}

	p.modelPath = modelPath
	p.logger.WithFields(logrus.Fields{
		"binary": llamaCppPath,
		"model":  modelPath,
		"port":   p.serverPort,
	}).Info("Starting llama.cpp server")

	// Prepare command arguments for LFM2 optimization
	args := []string{
		"--server",
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(p.serverPort),
		"--model", modelPath,
		"--n-predict", "4096",
		"--ctx-size", "8192",
		"--threads", "4", // Optimize for RPi5
		"--n-gpu-layers", "0", // CPU-only for RPi5
		"--chat-template", "chatml", // Use ChatML template for LFM2
		"--log-format", "json",
		"--verbose",
	}

	// Create command
	p.processCmd = exec.CommandContext(ctx, llamaCppPath, args...)

	// Set up environment
	p.processCmd.Env = os.Environ()

	// Create pipes for stdout/stderr
	stdout, err := p.processCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := p.processCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := p.processCmd.Start(); err != nil {
		return fmt.Errorf("failed to start llama.cpp server: %w", err)
	}

	// Monitor process output
	go p.monitorOutput(stdout, stderr)

	p.logger.WithField("pid", p.processCmd.Process.Pid).Info("llama.cpp server process started")
	return nil
}

// findLlamaCppBinary searches for llama.cpp binary in common locations
func (p *LlamaCppProvider) findLlamaCppBinary() (string, error) {
	p.logger.WithField("llamacpp_config", p.llamaCppConfig).Debug("Looking for llama.cpp binary")

	// Check configured binary path first
	if p.llamaCppConfig.BinaryPath != "" {
		p.logger.WithField("path", p.llamaCppConfig.BinaryPath).Info("Checking configured binary path")
		if _, err := os.Stat(p.llamaCppConfig.BinaryPath); err == nil {
			p.logger.WithField("path", p.llamaCppConfig.BinaryPath).Info("Found configured binary")
			return p.llamaCppConfig.BinaryPath, nil
		}
		p.logger.WithField("path", p.llamaCppConfig.BinaryPath).Warn("Configured binary path not found, falling back to default paths")
	} else {
		p.logger.Warn("No binary path configured, using default search paths")
	}

	possiblePaths := []string{
		"/usr/local/bin/llama-server",
		"/usr/bin/llama-server",
		"./llama-server",
		"/opt/llama.cpp/llama-server",
		"/home/pma/llama.cpp/llama-server",
		"/opt/pma/llama.cpp/build/bin/llama-server", // Add our actual path
	}

	p.logger.WithField("possible_paths", possiblePaths).Info("Searching for llama.cpp binary in default locations")

	for _, path := range possiblePaths {
		p.logger.WithField("checking_path", path).Debug("Checking binary path")
		if _, err := os.Stat(path); err == nil {
			p.logger.WithField("found_path", path).Info("Found llama.cpp binary")
			return path, nil
		}
	}

	return "", fmt.Errorf("llama.cpp binary not found in any of the expected locations")
}

// findModelFile searches for LFM2 model file
func (p *LlamaCppProvider) findModelFile() (string, error) {
	// Check configured model path first
	if p.llamaCppConfig.ModelPath != "" {
		if _, err := os.Stat(p.llamaCppConfig.ModelPath); err == nil {
			return p.llamaCppConfig.ModelPath, nil
		}
		p.logger.WithField("path", p.llamaCppConfig.ModelPath).Warn("Configured model path not found, falling back to default search")
	}

	// Check if model path is absolute
	if filepath.IsAbs(p.defaultModel) {
		if _, err := os.Stat(p.defaultModel); err == nil {
			return p.defaultModel, nil
		}
	}

	// Search in common model directories
	modelDirs := []string{
		"./models",
		"/opt/pma/models",
		"/home/pma/models",
		"/usr/local/share/models",
		"/opt/pma/llama.cpp/models", // Add our actual model directory
	}

	possibleNames := []string{
		p.defaultModel,
		p.defaultModel + ".gguf",
		"LFM2-1.2B.gguf",
		"LFM2-700M-Q4_K_M.gguf",
		"lfm2/LFM2-1.2B-Q4_K_M.gguf",
		"lfm2-full/LFM2-1.2B.gguf",
	}

	for _, dir := range modelDirs {
		for _, name := range possibleNames {
			fullPath := filepath.Join(dir, name)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath, nil
			}
		}
	}

	return "", fmt.Errorf("model file not found: %s", p.defaultModel)
}

// monitorOutput monitors the llama.cpp process output
func (p *LlamaCppProvider) monitorOutput(stdout, stderr io.ReadCloser) {
	// Monitor stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			p.logger.WithField("source", "llama.cpp-stdout").Debug(line)

			// Check for ready signal
			if strings.Contains(line, "HTTP server listening") {
				select {
				case p.processReady <- true:
				default:
				}
			}
		}
	}()

	// Monitor stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			p.logger.WithField("source", "llama.cpp-stderr").Debug(line)
		}
	}()
}

// waitForReady waits for the llama.cpp server to be ready
func (p *LlamaCppProvider) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for llama.cpp server to be ready")
		case <-p.processReady:
			// Double-check with health check
			if err := p.healthCheck(ctx); err == nil {
				p.isRunning = true
				p.logger.Info("llama.cpp server is ready")
				return nil
			}
		case <-ticker.C:
			if err := p.healthCheck(ctx); err == nil {
				p.isRunning = true
				p.logger.Info("llama.cpp server is ready")
				return nil
			}
		}
	}
}

// healthCheck checks if the llama.cpp server is running
func (p *LlamaCppProvider) healthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	p.lastHealthCheck = time.Now()
	return nil
}

// Shutdown stops the llama.cpp server if it was started by this provider
func (p *LlamaCppProvider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.processCmd != nil && p.processCmd.Process != nil {
		p.logger.Info("Stopping llama.cpp server")
		if err := p.processCmd.Process.Kill(); err != nil {
			p.logger.WithError(err).Warn("Failed to kill llama.cpp process")
		}
		p.processCmd.Wait()
		p.processCmd = nil
	}

	p.isRunning = false
	return nil
}

// GetName returns the provider name
func (p *LlamaCppProvider) GetName() string {
	return p.name
}

// IsAvailable checks if the provider is available
func (p *LlamaCppProvider) IsAvailable(ctx context.Context) bool {
	return p.healthCheck(ctx) == nil
}

// GetModels returns available models
func (p *LlamaCppProvider) GetModels(ctx context.Context) ([]ai.ModelInfo, error) {
	// For llama.cpp, we return the current loaded model
	models := []ai.ModelInfo{
		{
			ID:          p.defaultModel,
			Name:        p.defaultModel,
			Description: "LFM2 full precision model running on llama.cpp",
			Provider:    "llamacpp",
			Capabilities: []string{
				"chat",
				"completion",
				"tool_calling",
				"chatml_template",
			},
			MaxTokens:  8192,
			Available:  p.isRunning,
			LocalModel: true,
		},
	}

	return models, nil
}

// GetStats returns provider statistics
func (p *LlamaCppProvider) GetStats() ai.ProviderStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var avgResponseTime time.Duration
	if p.requestCount > 0 {
		avgResponseTime = time.Duration(int64(p.totalResponseTime) / p.requestCount)
	}

	return ai.ProviderStats{
		RequestCount:        p.requestCount,
		ErrorCount:          p.errorCount,
		AverageResponseTime: avgResponseTime,
		LastUsed:            p.lastHealthCheck,
	}
}

// Complete implements text completion (basic wrapper around chat)
func (p *LlamaCppProvider) Complete(ctx context.Context, prompt string, opts ai.CompletionOptions) (*ai.CompletionResponse, error) {
	// Convert completion request to chat format
	messages := []ai.ChatMessage{
		{
			Role:      "user",
			Content:   prompt,
			Timestamp: time.Now(),
		},
	}

	chatOpts := ai.ChatOptions{
		Provider:    opts.Provider,
		Model:       opts.Model,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		TopP:        opts.TopP,
		Stream:      opts.Stream,
	}

	// Use chat interface
	chatResp, err := p.Chat(ctx, messages, chatOpts)
	if err != nil {
		return nil, err
	}

	// Convert response
	return &ai.CompletionResponse{
		ID:               chatResp.ID,
		Text:             chatResp.Message.Content,
		FinishReason:     chatResp.FinishReason,
		Model:            chatResp.Model,
		Provider:         chatResp.Provider,
		ProcessingTimeMs: chatResp.ProcessingTimeMs,
		TokensUsed:       chatResp.TokensUsed,
		CreatedAt:        chatResp.CreatedAt,
	}, nil
}

// EstimateTokens provides a rough token count estimate
func (p *LlamaCppProvider) EstimateTokens(text string) int {
	// Rough estimate: ~4 characters per token for English text
	return len(text) / 4
}

// GetRateLimit returns rate limit information
func (p *LlamaCppProvider) GetRateLimit() ai.RateLimit {
	// llama.cpp doesn't have explicit rate limits, set reasonable defaults
	return ai.RateLimit{
		RequestsPerMinute: 60,    // Conservative for local processing
		TokensPerMinute:   10000, // Depends on hardware
	}
}

// HealthCheck implements health checking
func (p *LlamaCppProvider) HealthCheck(ctx context.Context) error {
	return p.healthCheck(ctx)
}
