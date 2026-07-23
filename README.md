# Telar

Telar is an open-source community platform being rebuilt around a Go
application API, a Next.js web client and a separate applied-AI service.

This repository continues the earlier Telar Social work with a new architecture
and a gradual migration path instead of presenting the rebuild as a finished
product.

> **Project status:** Telar is under active development. The main application,
> web client, AI Engine and client packages are present in the public
> repository, but the platform does not have a stable production release yet.

[Project website](https://telar.dev) ·
[Earlier Telar Social projects](https://telar.dev/legacy) ·
[Issues](https://github.com/Qolzam/telar/issues)

## What is in the repository

### Go application API

[`apps/api`](./apps/api) contains the main application backend. It is organized
by domain and currently includes work for:

- authentication and account verification;
- profiles and administration;
- posts, comments, votes and bookmarks;
- storage and shared application services;
- PostgreSQL persistence, caching, middleware and migrations;
- monolith and service-specific entry points for development and testing.

The migration is still in progress. Some domains are more complete and better
tested than others, so the repository should not yet be treated as a stable
public API.

### Next.js web client

[`apps/web`](./apps/web) is a Next.js and TypeScript application that uses the
workspace SDK. It contains authentication, account, profile, post, comment,
bookmark and administration features.

The web client is functional development code, but its product flow and visual
system are still being rebuilt.

### AI Engine

[`apps/ai-engine`](./apps/ai-engine) is a separate Go service for the
application and platform work around AI models. Its pushed code includes:

- document ingestion and Retrieval-Augmented Generation (RAG);
- configurable embedding and completion providers;
- Ollama and hosted-model adapters;
- Weaviate vector storage;
- content generation;
- prompt, chunking and content-analysis components;
- local Docker configuration and a browser demonstration.

AI-assisted moderation is being integrated across the AI Engine, application
API and administration interface. That work is active development, not a claim
of a complete or autonomous production moderation system.

### Shared packages and contracts

- [`packages/sdk`](./packages/sdk): TypeScript client used by the web
  application.
- [`packages/clients`](./packages/clients): Go clients for communication
  between application services.
- [`protos`](./protos): protobuf contracts for service boundaries.
- [`deployments/docker-compose`](./deployments/docker-compose): local
  application dependencies and backend services.
- [`tools`](./tools): development, migration, test and verification scripts.

## Repository structure

```text
telar/
├── apps/
│   ├── api/                 # Go application API and domain services
│   ├── web/                 # Next.js and TypeScript web client
│   └── ai-engine/           # Go RAG, generation and moderation service
├── packages/
│   ├── sdk/                 # TypeScript API client
│   └── clients/             # Go service clients
├── protos/                  # Service contracts
├── deployments/
│   └── docker-compose/      # Local backend environment
├── tools/                   # Development and verification scripts
├── docs/
└── Makefile
```

## Run the application locally

The current local workflow starts the database dependencies, Go API and
Next.js web client. The AI Engine has a separate setup described in its own
README.

### Requirements

- Go 1.24
- Node.js and pnpm
- Docker with Docker Compose
- GNU Make

### 1. Install the web dependencies

From the repository root:

```bash
pnpm install
```

### 2. Configure the application API

```bash
cp apps/api/.env.example apps/api/.env
```

The example values are for local development. Replace secrets and external
service configuration before using the application outside a local
environment.

### 3. Start the API and web client

```bash
make run-both
```

The command starts the local PostgreSQL and MailHog dependencies before running
the API and web development servers:

- API: `http://localhost:9099`
- Web client: `http://localhost:4000`
- MailHog: `http://localhost:8025`

Run the components separately when needed:

```bash
make run-api
make run-web
```

### 4. Run the AI Engine

The AI Engine uses its own environment and Docker Compose configuration:

```bash
cp apps/ai-engine/deployments/docker-compose/.env.example \
  apps/ai-engine/deployments/docker-compose/.env
```

Continue with the provider and startup instructions in the
[AI Engine README](./apps/ai-engine/README.md).

## Tests and quality checks

The Makefile contains focused tests for the Go application domains and the
release verification workflow.

Examples:

```bash
make test-auth
make test-posts
make test-comments
make test-e2e SERVICE=auth
make verify-release
```

For the web client:

```bash
cd apps/web
pnpm lint
pnpm build
pnpm test:e2e
```

For the AI Engine application packages:

```bash
cd apps/ai-engine
go test ./cmd/... ./internal/...
```

Some integration and end-to-end tests require the local database services and
environment configuration.

## Technology

- **Backend:** Go, PostgreSQL, Redis-compatible caching, REST and gRPC service
  boundaries
- **Web:** Next.js, React, TypeScript, Material UI and TanStack Query
- **Applied AI:** RAG, LangChainGo, Weaviate, Ollama and hosted LLM APIs
- **Local development:** Docker Compose, Make and pnpm workspaces
- **Testing:** Go tests, Playwright and repository verification scripts

## Current boundaries

The repository does not currently provide:

- a stable production release or customer-facing availability commitment;
- a completed migration of every earlier Telar Social capability;
- a single Docker command that starts the web application, API and AI Engine
  together;
- published Kubernetes manifests for the rebuilt platform;
- verified production-scale benchmarks for the rebuilt system;
- a managed Telar hosting service.

Kubernetes deployment, wider AI search and managed platform capabilities are
development directions. They should not be described as shipped until their
code, setup instructions and validation results are public.

## Development direction

Current work is focused on:

1. completing the application-domain migration to the Go and PostgreSQL
   architecture;
2. stabilizing the Next.js client and TypeScript SDK;
3. integrating the AI Engine through explicit application and service
   boundaries;
4. improving moderation review, tenant separation and failure handling;
5. making the full local environment reproducible from a clean checkout;
6. publishing deployment guidance and measurements after validation.

Milestones should be marked complete only when their code, setup instructions
and tests are available in the repository.

## Contributing

Telar is being developed in public. Before opening a change, read
[CONTRIBUTING.md](./CONTRIBUTING.md) and check the
[open issues](https://github.com/Qolzam/telar/issues).

## License

Telar is available under the [MIT License](./LICENSE).
