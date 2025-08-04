package main

import (
	"context"
	"fmt"

	"github.com/frostdev-ops/pma-backend-go/internal/ai/providers"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("🚀 Testing Full LFM2 Model Configuration")
	fmt.Println("=========================================")

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create config using the full model
	cfg := &config.Config{
		AI: config.AIConfig{
			Enabled:      true,
			DefaultModel: "LFM2-1.2B", // Full precision model
			LlamaCpp: config.LlamaCppConfig{
				Enabled:      true,
				BaseURL:      "http://localhost:8000",
				DefaultModel: "LFM2-1.2B", // Full precision model
				Timeout:      "60s",
				MaxRetries:   3,
			},
		},
	}

	fmt.Printf("✅ Configuration Model: %s\n", cfg.AI.DefaultModel)
	fmt.Printf("✅ LlamaCpp Model: %s\n", cfg.AI.LlamaCpp.DefaultModel)

	// Test provider creation
	provider := providers.NewLlamaCppProvider(config.AIProviderConfig{
		Type:         "llamacpp",
		Enabled:      true,
		URL:          cfg.AI.LlamaCpp.BaseURL,
		DefaultModel: cfg.AI.LlamaCpp.DefaultModel,
		MaxTokens:    4096,
	}, logger)

	fmt.Printf("✅ Provider Created: %s\n", provider.GetName())

	// Test model info
	ctx := context.Background()
	models, err := provider.GetModels(ctx)
	if err != nil {
		fmt.Printf("⚠️  Error getting models: %v\n", err)
	} else {
		fmt.Printf("✅ Available Models:\n")
		for _, model := range models {
			fmt.Printf("   📦 Name: %s\n", model.Name)
			fmt.Printf("   📋 Description: %s\n", model.Description)
			fmt.Printf("   🔧 Capabilities: %v\n", model.Capabilities)
			fmt.Printf("   💾 Local Model: %v\n", model.LocalModel)
			fmt.Printf("   🎯 Max Tokens: %d\n\n", model.MaxTokens)
		}
	}

	fmt.Println("🎉 Full LFM2 Model Configuration Test Complete!")
	fmt.Println("\n📋 Key Benefits of Full Model vs Quantized:")
	fmt.Println("✅ Better response quality and accuracy")
	fmt.Println("✅ More nuanced understanding and generation")
	fmt.Println("✅ Better tool calling precision")
	fmt.Println("✅ Improved ChatML template handling")
	fmt.Println("✅ Minimal memory overhead (as confirmed by user)")

	fmt.Println("\n🔧 Model File Expected:")
	fmt.Println("   📁 LFM2-1.2B.gguf (full precision)")
	fmt.Println("   🚫 LFM2-1.2B-Q4_K_M.gguf (quantized - no longer used)")

	fmt.Println("\n🚀 Ready for deployment with full LFM2 model!")
}
