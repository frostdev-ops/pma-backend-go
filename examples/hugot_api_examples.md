# Hugot API Configuration Examples

This document demonstrates how to configure and use Hugot via API calls. All settings are automatically saved to the database.

## Available API Endpoints

### 1. Get Hugot Settings
```bash
curl -X GET http://localhost:3001/api/v1/ai/hugot/settings
```

**Response:**
```json
{
  "hugot_settings": {
    "enabled": true,
    "models_dir": "./models/hugot",
    "default_model": "LiquidAI/LFM2-1.2B",
    "timeout": "30s",
    "max_retries": 3,
    "available_models": [
      "LiquidAI/LFM2-1.2B",
      "feature-extraction",
      "text-classification"
    ],
    "status": "connected"
  }
}
```

### 2. Update Hugot Settings
```bash
curl -X PUT http://localhost:3001/api/v1/ai/hugot/settings \
  -H 'Content-Type: application/json' \
  -d '{
    "enabled": true,
    "models_dir": "./models/hugot",
    "default_model": "LiquidAI/LFM2-1.2B",
    "timeout": "45s",
    "max_retries": 5
  }'
```

**Response:**
```json
{
  "message": "Hugot settings updated successfully",
  "settings": {
    "enabled": true,
    "models_dir": "./models/hugot", 
    "default_model": "LiquidAI/LFM2-1.2B",
    "timeout": "45s",
    "max_retries": 5
  }
}
```

### 3. Test Hugot Connection
```bash
curl -X POST http://localhost:3001/api/v1/ai/hugot/test-connection \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "LiquidAI/LFM2-1.2B"
  }'
```

**Response:**
```json
{
  "test_result": {
    "success": true,
    "message": "Connection successful",
    "latency": "15ms",
    "tested_at": "2024-01-15T10:30:00Z"
  }
}
```

### 4. Get Available Models
```bash
curl -X GET http://localhost:3001/api/v1/ai/hugot/models
```

**Response:**
```json
{
  "models": [
    {
      "id": "LiquidAI/LFM2-1.2B",
      "name": "LiquidAI LFM2 1.2B",
      "description": "1.2B parameter language model optimized for efficiency",
      "provider": "hugot",
      "max_tokens": 4096,
      "available": true,
      "local_model": true,
      "capabilities": ["text-generation", "chat", "completion"]
    },
    {
      "id": "feature-extraction",
      "name": "Feature Extraction",
      "description": "Extract semantic embeddings from text",
      "provider": "hugot",
      "max_tokens": 512,
      "available": true,
      "local_model": true,
      "capabilities": ["embeddings", "similarity"]
    }
  ],
  "count": 2
}
```

## Configuration Features

### Database Storage
- All Hugot settings are automatically saved to the database
- Settings persist across application restarts
- Configuration changes are logged for audit purposes

### Supported Models
- **LiquidAI/LFM2-1.2B**: Full language model for text generation and chat
- **feature-extraction**: Semantic embeddings for RAG and similarity
- **text-classification**: Text categorization and sentiment analysis

### Model Capabilities
Each model supports different capabilities:
- **text-generation**: Complete text prompts
- **chat**: Conversational interactions
- **completion**: Text completion tasks
- **embeddings**: Vector embeddings for similarity
- **classification**: Text categorization

## Integration with Existing AI System

### Provider Fallback
Hugot integrates seamlessly with the existing AI provider system:
- Ollama remains the primary provider for most tasks
- Hugot provides local transformer capabilities
- Automatic fallback between providers based on availability

### Unified API
Use the standard AI endpoints with Hugot:
```bash
# Text completion
curl -X POST http://localhost:3001/api/v1/ai/completion \
  -H 'Content-Type: application/json' \
  -d '{
    "prompt": "The future of AI is",
    "model": "LiquidAI/LFM2-1.2B",
    "provider": "hugot",
    "max_tokens": 100
  }'

# Chat
curl -X POST http://localhost:3001/api/v1/ai/chat \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "model": "LiquidAI/LFM2-1.2B",
    "provider": "hugot"
  }'
```

## Model Management

### Download Models
Models are automatically downloaded to the configured `models_dir` when first used:
```bash
# Enable auto-download
curl -X PUT http://localhost:3001/api/v1/ai/hugot/settings \
  -d '{"models_dir": "./models/hugot", "enabled": true}'
```

### Custom Models
Add custom ONNX models by placing them in the models directory:
```bash
# Place your model in ./models/hugot/my-custom-model/
# Then configure it:
curl -X PUT http://localhost:3001/api/v1/ai/hugot/settings \
  -d '{"default_model": "my-custom-model"}'
```

## Performance Tuning

### Timeout Configuration
Adjust timeouts based on model size and hardware:
```bash
# For larger models, increase timeout
curl -X PUT http://localhost:3001/api/v1/ai/hugot/settings \
  -d '{"timeout": "120s"}'
```

### Retry Configuration
Configure retry behavior for better reliability:
```bash
curl -X PUT http://localhost:3001/api/v1/ai/hugot/settings \
  -d '{"max_retries": 3}'
```

## Local vs Cloud Processing

### Benefits of Hugot
- **Privacy**: All processing happens locally
- **Speed**: No network latency for inference
- **Cost**: No API costs for model usage
- **Reliability**: Works offline

### Use Cases
- **Embeddings**: For RAG systems and semantic search
- **Classification**: Content categorization and filtering  
- **Small Language Models**: Fast text generation for specific domains
- **Real-time Processing**: Low-latency AI tasks

## Error Handling

### Common Errors
1. **Model Not Found**: Check model directory and permissions
2. **Out of Memory**: Use smaller models or increase system memory
3. **Timeout**: Increase timeout setting for larger models
4. **Connection Failed**: Verify Hugot service is running

### Debugging
Enable debug logging to troubleshoot issues:
```bash
# Check provider status
curl -X GET http://localhost:3001/api/v1/ai/providers

# Test connection
curl -X POST http://localhost:3001/api/v1/ai/hugot/test-connection
```