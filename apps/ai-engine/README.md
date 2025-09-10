# AI Engine

## 🎯 Core Responsibility

The AI Engine is a standalone Go microservice responsible for all Retrieval-Augmented Generation (RAG) and Large Language Model (LLM) operations within the Telar platform.

Its primary purpose is to decouple the core social platform from the complexities of AI orchestration, allowing for independent scaling, development, and flexible integration with various AI providers. It provides a simple, stateless API for ingesting knowledge and answering questions based on that knowledge.

---

## 🛠️ Tech Stack & Design

This service is built with a focus on performance, flexibility, and production-readiness.

*   **Language:** Go (Golang)
*   **Web Framework:** Fiber 
*   **Configuration:** Viper (from environment variables)
*   **Vector Database Client:** Weaviate Go Client
*   **Core Architectural Pattern:** **Provider-Agnostic LLM Abstraction**
    *   The engine is built around a central Go `interface` (`platform/llm/client.go`).
    *   This allows the service to seamlessly switch between different LLM backends via a single configuration change, without altering the core business logic.
    *   The Phase 1 implementation uses **Ollama** to run open-source models locally for cost-effective and private development.
