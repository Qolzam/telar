# AI Engine

## 🎯 Core Responsibility

The AI Engine is a standalone Go microservice responsible for all Retrieval-Augmented Generation (RAG) and Large Language Model (LLM) operations within the Telar platform.

Its primary purpose is to decouple the core social platform from the complexities of AI orchestration, allowing for independent scaling, development, and flexible integration with various AI providers. It provides a simple, stateless API for ingesting knowledge and answering questions based on that knowledge.

---

## 🏁 Getting Started

This service is designed to run locally using Docker and Ollama.

### Prerequisites

1.  **Docker & Docker Compose:** [Install Docker](https://docs.docker.com/get-docker/)
2.  **Ollama:** [Install Ollama](https://ollama.com/)

### 1. Pull the Necessary AI Models

For the RAG pipeline to function, you need two types of models: one for generating embeddings and one for generating chat responses.

**Important**: Ollama must be running to pull models and to use the AI features:

```bash
# Start Ollama service first
ollama serve
```

Then in a new terminal, pull the required models:

```bash
# Pull the recommended embedding model
ollama pull nomic-embed-text

# Pull the recommended chat model
ollama pull llama3:8b
```

### 2. Configure Environment Variables

Copy the example environment file and modify as needed:

```bash
cd apps/ai-engine/deployments/docker-compose/
cp .env.example .env
```

Key environment variables for the AI Engine:

```ini
# Server Configuration
PORT=8000
HOST=0.0.0.0

# LLM Provider Configuration
LLM_PROVIDER=ollama
OLLAMA_BASE_URL=http://host.docker.internal:11434
EMBEDDING_MODEL=nomic-embed-text
COMPLETION_MODEL=llama3:8b

# Vector Database Configuration
WEAVIATE_URL=http://weaviate:8080
WEAVIATE_API_KEY=
```

### 3. Run the Service

You can run the AI Engine and its required database from the docker-compose directory:

```bash
cd apps/ai-engine/deployments/docker-compose/
docker compose up
```

The AI Engine API will now be available at `http://localhost:8000`.

---

## ⚙️ API Usage

Interact with the engine using its REST API. All endpoints are prefixed with `/api/v1`.

### Health Check

Check if the service and its dependencies are running correctly.

*   **Endpoint:** `GET /health`
*   **Example `curl`:**
    ```bash
    curl http://localhost:8000/health
    ```

### Ingest Content

Add a document to the knowledge base.

*   **Endpoint:** `POST /api/v1/ingest`
*   **Body:**
    ```json
    {
      "text": "The Telar platform is an open-source social network built with Go and Next.js.",
      "metadata": {
        "source": "documentation",
        "type": "platform_info"
      }
    }
    ```

*   **Example `curl`:**
    ```bash
    curl -X POST http://localhost:8000/api/v1/ingest \
    -H "Content-Type: application/json" \
    -d '{"text": "The Telar platform is an open-source social network built with Go and Next.js.", "metadata": {"source": "documentation"}}'
    ```

*   **Successful Response:**
    ```json
    {
      "status": "success",
      "message": "Document ingested successfully",
      "id": "doc-abc123..."
    }
    ```

### Query the Knowledge Base

Ask a question and get a synthesized answer based on the ingested content.

*   **Endpoint:** `POST /api/v1/query`
*   **Body:**
    ```json
    {
      "question": "What is Telar built with?",
      "limit": 5,
      "context": {
        "user_intent": "technical_inquiry"
      }
    }
    ```

*   **Example `curl`:**
    ```bash
    curl -X POST http://localhost:8000/api/v1/query \
    -H "Content-Type: application/json" \
    -d '{"question": "What is Telar built with?"}'
    ```

*   **Successful Response:**
    ```json
    {
      "answer": "The Telar platform is built using Go and Next.js.",
      "sources": [
        {
          "id": "doc-abc123...",
          "text": "The Telar platform is an open-source social network built with Go and Next.js.",
          "score": 0.95,
          "metadata": {
            "source": "documentation"
          }
        }
      ]
    }
    ```

---

## 🛠️ Tech Stack & Design

This service is built with a focus on performance, flexibility, and production-readiness.

*   **Language:** Go (Golang)
*   **Web Framework:** Fiber
*   **Configuration:** Viper (from environment variables)
*   **Vector Database Client:** Weaviate Go Client
*   **Core Architectural Pattern:** **Provider-Agnostic LLM Abstraction**
    *   The engine is built around a central Go `interface` defined in `internal/platform/llm/client.go`.
    *   This allows the service to seamlessly switch between different LLM backends via a single configuration change (`LLM_PROVIDER`), without altering the core business logic.

---

## 🗺️ Project Roadmap

This project is being developed in deliberate phases to ensure a robust and feature-complete architecture.

-   [x] **Phase 1: Local-First Foundation**
    -   Implement manual RAG pipeline.
    -   Build the provider-agnostic `llm.Client` interface.
    -   Implement the **Ollama** client for local, cost-free development.

-   [ ] **Phase 2: High-Performance Showcase**
    -   Implement the **Groq** client for high-speed cloud inference.
    -   Create a "wow" demo showcasing near-instant RAG responses.

-   [ ] **Phase 3: Enterprise-Ready Refactor**
    -   Implement the **OpenAI** client for commercial API compatibility.
    -   Refactor the manual RAG orchestration to use **LangChainGo**.
    -   Add comprehensive unit tests with mocked interfaces.
