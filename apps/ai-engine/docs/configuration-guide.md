# AI Engine Configuration Guide

## 🎯 **Professional Configuration Management**

The AI Engine features a **fully provider-agnostic architecture** with independently configurable backends for both embedding and completion tasks. This guide explains how to configure the service for different deployment scenarios.

## 📋 **Quick Start Scenarios**

### **Scenario 1: Fully Local Development** (Cost: $0)
```bash
EMBEDDING_PROVIDER=ollama
COMPLETION_PROVIDER=ollama
OLLAMA_BASE_URL=http://localhost:11434
EMBEDDING_MODEL=nomic-embed-text
COMPLETION_MODEL=llama3:8b
```
**Use Case**: Development, testing, and full privacy.

### **Scenario 2: High-Speed Prototyping** (Cost: Low)
```bash
EMBEDDING_PROVIDER=ollama
COMPLETION_PROVIDER=groq
GROQ_API_KEY=your-groq-api-key
GROQ_MODEL=llama-3.1-8b-instant
OLLAMA_BASE_URL=http://localhost:11434
EMBEDDING_MODEL=nomic-embed-text
```
**Use Case**: The best "wow" demo experience. Blazing fast answers.

### **Scenario 3: Enterprise Cloud-Native** (Cost: High)
```bash
EMBEDDING_PROVIDER=openai
COMPLETION_PROVIDER=openai
OPENAI_API_KEY=your-openai-api-key
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
OPENAI_COMPLETION_MODEL=gpt-3.5-turbo
```
**Use Case**: Production deployments requiring a fully managed, auditable cloud pipeline.

### **Scenario 4: Mixed Enterprise** (Cost: High)
```bash
EMBEDDING_PROVIDER=openai
COMPLETION_PROVIDER=groq
OPENAI_API_KEY=your-openai-api-key
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
GROQ_API_KEY=your-groq-api-key
GROQ_MODEL=llama-3.1-8b-instant
```
**Use Case**: Production deployments that need OpenAI's embedding quality but Groq's completion speed.

## 🔧 **Configuration Variables Reference**

### **Core Provider Selection**
- `EMBEDDING_PROVIDER`: `ollama` | `openai` ⚠️ *Note: Groq and OpenRouter do not support embeddings*
- `COMPLETION_PROVIDER`: `ollama` | `groq` | `openai` | `openrouter`

### **Ollama Settings** (Local Models)
- `OLLAMA_BASE_URL`: Ollama server URL (default: `http://localhost:11434`)
- `EMBEDDING_MODEL`: Embedding model name (default: `nomic-embed-text`)
- `COMPLETION_MODEL`: Completion model name (default: `llama3:8b`)

### **Groq Settings** (High-Speed Inference - Completions Only)
- `GROQ_API_KEY`: Your Groq API key
- `GROQ_MODEL`: Model name (e.g., `llama-3.1-8b-instant`)
- ⚠️ **Note**: Groq does not support embeddings - use for completions only

### **OpenAI Settings** (Enterprise Compatibility)
- `OPENAI_API_KEY`: Your OpenAI API key
- `OPENAI_EMBEDDING_MODEL`: Embedding model (default: `text-embedding-3-small`)
- `OPENAI_COMPLETION_MODEL`: Completion model (default: `gpt-3.5-turbo`)

### **OpenRouter Settings** (Cost-Effective Testing - Completions Only)
- **Note**: OpenRouter uses OpenAI compatibility - no separate API key needed
- `OPENAI_API_KEY`: Your OpenRouter API key (same as OpenAI)
- `OPENAI_BASE_URL`: Set to `https://openrouter.ai/api/v1`
- `OPENAI_MODEL`: Model name (e.g., `anthropic/claude-3-haiku`)
- ⚠️ **Note**: OpenRouter does not support embeddings - use for completions only

### **Infrastructure Settings**
- `WEAVIATE_URL`: Vector database URL (default: `http://weaviate:8080`)
- `WEAVIATE_API_KEY`: Vector database API key (optional)
- `SERVER_PORT`: Service port (default: `8000`)
- `SERVER_ENV`: Environment (default: `development`)

## ✅ **Configuration Validation**

The AI Engine includes robust startup validation that will:

1. **Validate Provider Selection**: Ensure only supported providers are specified
2. **Check Required API Keys**: Verify API keys are provided for selected providers
3. **Provide Clear Error Messages**: Help you fix configuration issues quickly

### **Example Error Messages**
```bash
❌ Configuration validation failed: invalid EMBEDDING_PROVIDER: 'groq'. Supported providers are 'ollama', 'openai'

❌ Configuration validation failed: COMPLETION_PROVIDER is 'groq' but GROQ_API_KEY is not set

⚠️ Embeddings are currently not supported by Groq. Please use Ollama or OpenAI for embeddings, or use a hybrid configuration with Groq for completions only
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
