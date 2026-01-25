
# AI Engine

## 🎯 Core Responsibility

The AI Engine is a standalone Go microservice responsible for all Large Language Model (LLM) operations within the Telar platform, including Retrieval-Augmented Generation (RAG).

Its primary purpose is to decouple the core social platform from the complexities of AI orchestration. This allows for independent scaling, development, and a flexible, provider-agnostic architecture that can adapt to any deployment need, from fully local and private to enterprise-grade cloud native.

---

## 🏁 Getting Started

This guide provides the steps to get the AI Engine and its dependencies running locally with a single command.

### Prerequisites

1.  **Docker & Docker Compose:** [Install Docker](https://docs.docker.com/get-docker/)
2.  **Ollama:** [Install Ollama](https://ollama.com/) (Required for local development scenarios)

### 1. Configure Your Environment

First, copy the example environment file from the deployment directory to the repository root.

```bash
# Run this from the root of the 'telar' repository
cp apps/ai-engine/deployments/docker-compose/.env.example .env
```

Now, open the `.env` file and configure your desired AI providers. For a quick start, the default "Fully Local" scenario requires no changes. For other scenarios (like using Groq or OpenAI), you must add your API keys.

### 2. Run the Service

With your configuration in place, start the entire AI Engine stack.

**Option A: Quick Start (Recommended)**
```bash
# Run from the ai-engine directory - starts everything with one command
cd apps/ai-engine
./run_dev.sh start
```

**Option B: Manual Docker Compose**
```bash
# This command must be run from the root of the 'telar' repository
docker-compose -f apps/ai-engine/deployments/docker-compose/docker-compose.yml up --build -d
```

The AI Engine API will now be available at `http://localhost:9066`.

### 3. Test the System

**Option A: Interactive Demo UI**
```bash
# Open your browser and navigate to:
http://localhost:9066
```


**Option B: Direct API Testing**

```bash
# Health check
curl http://localhost:9066/health

# Ingest a document
curl -X POST http://localhost:9066/api/v1/ingest \
  -H "Content-Type: application/json" \
  -d '{"text": "The Telar platform is built with Go and Next.js.", "metadata": {"source": "docs"}}'

# Query the knowledge base
curl -X POST http://localhost:9066/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"question": "What is Telar built with?"}'
```

---

## 🏛️ Architecture & Tech Stack

This service is architected for performance, flexibility, and production-readiness.

*   **Language:** Go (Golang)
*   **Web Framework:** Fiber
*   **LLM Orchestration:** LangChainGo for prompt management and LLM abstraction
*   **Core Architectural Pattern:** **Fully Provider-Agnostic LLM Architecture**
    *   The engine features **independently configurable backends** for both embedding and completion tasks, allowing a user to mix and match providers based on specific needs (cost, performance, privacy).
    *   **Embedding Providers:** Ollama (local), OpenAI (cloud)
    *   **Completion Providers:** Ollama (local), Groq (high-speed cloud), OpenAI (enterprise cloud), OpenRouter (cost-effective cloud)

### In-Depth Architectural Documentation

For a complete breakdown of the system design, data flows, and deployment scenarios, please see our comprehensive documentation:

*   **[📄 Comprehensive Flow Analysis](./docs/comprehensive-flow-analysis.md)** - The complete technical analysis with data flow diagrams and performance matrices.
*   **[⚙️ Configuration Guide](./docs/configuration-guide.md)** - A step-by-step setup guide for all supported deployment scenarios.
*   **[🎨 Visual Flow Diagrams](./docs/visual-flow-diagrams.md)** - At-a-glance diagrams of the system architecture.

### Supported Deployment Scenarios

The AI Engine supports multiple deployment scenarios, each optimized for a different use case.

| Scenario | Embedding | Completion | Use Case | Performance | Cost |
|:---|:---|:---|:---|:---|:---|
| **Local Development** | Ollama | Ollama | Development & Testing | Medium | Free |
| **High-Speed Hybrid** | Ollama | Groq | Demos & Prototyping | Ultra-Fast | Low |
| **Enterprise Production** | OpenAI | OpenAI | Production Workloads | High | High |
| **Mixed Enterprise** | OpenAI | Groq | High-Speed Production | Ultra-Fast | High |

> **📚 For complete details on all scenarios, see our [Configuration Guide](./docs/configuration-guide.md)**.

---

## 🚀 Features

### 1. Knowledge Management (RAG)
- **Document Ingestion**: Store and vectorize documents for semantic search
- **Intelligent Query**: Ask questions and get contextual answers from your knowledge base
- **Provider-Agnostic**: Works with Ollama, OpenAI, Groq, or OpenRouter

### 2. Content Generation
- **Conversation Starters**: Generate engaging discussion prompts for communities
- **Concurrent Request Management**: Built-in rate limiting and queue management
- **Style Customization**: Generate content in different tones and styles

### 3. Content Moderation (NEW) ✨
- **AI-Powered Analysis**: Automatically analyze content for policy violations
- **Multi-Category Scoring**: Toxicity, sexual content, violence, spam, and misinformation
- **Structured Results**: Get detailed scores, confidence levels, and reasons
- **Async Processing**: Non-blocking analysis for high-performance applications

## 🛡️ Moderation Architecture: The Guardian Protocol

The AI Engine utilizes a **Tiered Pipeline Architecture** to balance cost, speed, and accuracy. It does not rely solely on expensive LLM calls.

1.  **L1 Cache (Instant):** SHA-256 hash lookup prevents re-analyzing known content. Latency: <1ms.
2.  **L2 Heuristics (Zero-Latency):** Regex and Keyword matching layer filters obvious violations (spam patterns, slurs) before they reach the inference layer. Latency: <5ms.
3.  **L3 Semantic Analysis (Deep Reasoning):** If content passes L1 and L2, it is analyzed by a Large Language Model (Qwen 2.5) for context-aware decisions. Latency: ~3s (CPU).

*Architectural Note: The L3 layer is designed behind a `ContentModerator` interface, allowing for hot-swapping to high-throughput ONNX/DistilBERT models for enterprise-scale deployments without changing the core pipeline logic.*

**Example Usage**:
```bash
# Analyze content for moderation
curl -X POST http://localhost:9066/api/v1/analyze/content \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{"content": "Text to analyze"}'

# Response
{
  "is_flagged": true,
  "flag_reason": "Toxicity exceeds 0.7",
  "scores": {
    "toxicity": 0.95,
    "sexual": 0.0,
    "violence": 0.0,
    "spam": 0.0,
    "misinformation": 0.0
  },
  "confidence": 0.98,
  "timestamp": "2025-10-09T12:00:00Z"
}
```

**Documentation**:
- [Analyzer Package README](./internal/analyzer/README.md)
- [Integration Guide](./docs/INTEGRATION_GUIDE.md)
- [Implementation Status](./docs/PHASE5_IMPLEMENTATION_STATUS.md)

---

## 🗺️ Project Roadmap

This project is being developed in deliberate phases to ensure a robust and feature-complete architecture.

-   [x] **Phase 1: Local-First Foundation** - Implemented a manual RAG pipeline with a provider-agnostic interface using Ollama.
-   [x] **Phase 2: High-Performance Showcase** - Integrated the Groq client and evolved the architecture to a specialized hybrid model (embeddings vs. completions).
-   [x] **Phase 3: Enterprise-Ready Refactor** - Integrated OpenAI, refactored orchestration to LangChainGo, and built a comprehensive, enterprise-grade testing suite.
-   [x] **Phase 4: Product-First Feature Launch** - Launched the "Community Ignition Toolkit" for generating high-quality conversation starters.
-   [x] **Phase 5: The Guardian Protocol** - Built AI-powered content moderation system for proactive community safety (Backend Complete).

---

## 🔧 Troubleshooting & Support

For common issues and solutions, please refer to the troubleshooting section in our detailed **[Configuration Guide](./docs/configuration-guide.md)**.

```