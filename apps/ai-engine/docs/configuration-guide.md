# AI Engine Configuration Guide

## 🎯 ** Configuration Management**


## 📋 **Quick Start Scenarios**

### **Scenario 1: Fully Local Development** (Cost: $0)
```bash
KNOWLEDGE_EMBEDDING_PROVIDER=ollama
KNOWLEDGE_EMBEDDING_MODEL=nomic-embed-text
GENERATOR_PROVIDER=ollama
GENERATOR_MODEL=llama3:8b
MODERATION_FALLBACK_PROVIDER=ollama
MODERATION_FALLBACK_MODEL=qwen2.5:1.5b
OLLAMA_BASE_URL=http://localhost:11434
```
**Use Case**: Development, testing, and full privacy.

### **Scenario 2: High-Speed Prototyping** (Cost: Low)
```bash
KNOWLEDGE_EMBEDDING_PROVIDER=ollama
KNOWLEDGE_EMBEDDING_MODEL=nomic-embed-text
GENERATOR_PROVIDER=groq
GENERATOR_MODEL=llama-3.1-8b-instant
MODERATION_FALLBACK_PROVIDER=groq
MODERATION_FALLBACK_MODEL=llama-3.1-8b-instant
GROQ_API_KEY=your-groq-api-key
OLLAMA_BASE_URL=http://localhost:11434
```
**Use Case**: The best "wow" demo experience. Blazing fast answers.

### **Scenario 3: Enterprise Cloud-Native** (Cost: High)
```bash
KNOWLEDGE_EMBEDDING_PROVIDER=openai
KNOWLEDGE_EMBEDDING_MODEL=text-embedding-3-small
GENERATOR_PROVIDER=openai
GENERATOR_MODEL=gpt-3.5-turbo
MODERATION_FALLBACK_PROVIDER=openai
MODERATION_FALLBACK_MODEL=gpt-3.5-turbo
OPENAI_API_KEY=your-openai-api-key
```
**Use Case**: Production deployments requiring a fully managed, auditable cloud pipeline.

### **Scenario 4: Mixed Enterprise** (Cost: High)
```bash
KNOWLEDGE_EMBEDDING_PROVIDER=openai
KNOWLEDGE_EMBEDDING_MODEL=text-embedding-3-small
GENERATOR_PROVIDER=groq
GENERATOR_MODEL=llama-3.1-8b-instant
MODERATION_FALLBACK_PROVIDER=groq
MODERATION_FALLBACK_MODEL=llama-3.1-8b-instant
OPENAI_API_KEY=your-openai-api-key
GROQ_API_KEY=your-groq-api-key
```
**Use Case**: Production deployments that need OpenAI's embedding quality but Groq's completion speed.

## 🔧 **Configuration Variables Reference**

### **Global Infrastructure**
- `OLLAMA_BASE_URL`: Ollama server URL (default: `http://localhost:11434`)
- `GROQ_API_KEY`: Your Groq API key
- `OPENAI_API_KEY`: Your OpenAI API key
- `OPENAI_BASE_URL`: OpenAI API base URL (default: `https://api.openai.com/v1`)
- `GLOBAL_ONNX_LIB_PATH`: Path to ONNX Runtime library (default: `./libs/libonnxruntime.so`)
- `MAX_CONCURRENT`: Maximum concurrent requests (default: `2`)

### **Feature: Knowledge (RAG)**
- `KNOWLEDGE_EMBEDDING_PROVIDER`: `ollama` | `openai` ⚠️ *Note: Groq and OpenRouter do not support embeddings*
- `KNOWLEDGE_EMBEDDING_MODEL`: Embedding model name (default: `nomic-embed-text`)

### **Feature: Generator**
- `GENERATOR_PROVIDER`: `ollama` | `groq` | `openai` | `openrouter`
- `GENERATOR_MODEL`: Model name (default: `llama3:8b`)

### **Feature: Moderation**
- `MODERATION_ONNX_TOXICITY_MODEL_PATH`: Path to toxicity ONNX model file (optional, default: `./models/distilbert/model.onnx`)
- `MODERATION_ONNX_SPAM_MODEL_PATH`: Path to spam ONNX model file (optional, default: `./models/spam/model.onnx`)
- `MODERATION_FALLBACK_PROVIDER`: `ollama` | `groq` | `openai` | `openrouter` (mandatory)
- `MODERATION_FALLBACK_MODEL`: Model name for moderation fallback (default: `qwen2.5:1.5b`)
- `MODERATION_TOXICITY_THRESHOLD`: Toxicity threshold (0.0-1.0, default: `0.50`)
- `MODERATION_SPAM_THRESHOLD`: Spam threshold (0.0-1.0, default: `0.45`)
- `MODERATION_SEXUAL_THRESHOLD`: Sexual content threshold (0.0-1.0, default: `0.75`)
- `MODERATION_VIOLENCE_THRESHOLD`: Violence threshold (0.0-1.0, default: `0.75`)
- `MODERATION_MISINFORMATION_THRESHOLD`: Misinformation threshold (0.0-1.0, default: `0.70`)

### **Provider-Specific Model Configuration**
- For Ollama: Models are specified via `KNOWLEDGE_EMBEDDING_MODEL`, `GENERATOR_MODEL`, `MODERATION_FALLBACK_MODEL`
- For Groq: Models are specified via `GENERATOR_MODEL` and `MODERATION_FALLBACK_MODEL` (Groq does not support embeddings)
- For OpenAI: Models are specified via `GENERATOR_MODEL` and `MODERATION_FALLBACK_MODEL` (embedding model via `KNOWLEDGE_EMBEDDING_MODEL`)
- For OpenRouter: Uses OpenAI compatibility - set `OPENAI_BASE_URL=https://openrouter.ai/api/v1` and use `OPENAI_API_KEY` with OpenRouter key

### **Infrastructure Settings**
- `WEAVIATE_URL`: Vector database URL (default: `http://weaviate:8080` for internal Docker network, `http://localhost:9077` for external access)
- `WEAVIATE_API_KEY`: Vector database API key (optional)
- `AI_ENGINE_PORT`: Service port (default: `9066`)
- `SERVER_ENV`: Environment (default: `development`)

## ✅ **Configuration Validation**

The AI Engine includes robust startup validation that will:

1. **Validate Provider Selection**: Ensure only supported providers are specified
2. **Check Required API Keys**: Verify API keys are provided for selected providers
3. **Provide Clear Error Messages**: Help you fix configuration issues quickly

### **Example Error Messages**
```bash
❌ Configuration validation failed: unsupported knowledge embedding provider: 'groq' (supported: ollama, openai)

❌ Configuration validation failed: GROQ_API_KEY is required when using Groq for generator

❌ Configuration validation failed: MODERATION_FALLBACK_PROVIDER is required (moderation fallback is mandatory)

⚠️ Embeddings are currently not supported by Groq. Please use Ollama or OpenAI for knowledge embeddings
```

## 🚀 **Best Practices**

1. **Start Simple**: Begin with Scenario 1 (Fully Local) for development
2. **Use Scenarios**: Follow the predefined scenarios for common use cases
3. **Validate Early**: The service will validate your configuration on startup
4. **Environment-Specific**: Use different configurations for dev/staging/production
5. **Secure Keys**: Never commit API keys to version control

## 🔍 **Troubleshooting**

### **Common Issues**
- **"Configuration validation failed"**: Check your provider selection and API keys
- **"Failed to create client"**: Verify your API keys are valid and have sufficient credits
- **"Connection refused"**: Ensure Ollama is running for local scenarios

### **Getting Help**
- Check the logs for detailed error messages
- Verify your API keys are correct and active
- Ensure all required services (Ollama, Weaviate) are running
