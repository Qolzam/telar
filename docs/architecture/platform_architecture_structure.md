# Telar Platform Architecture & Structure
*Updated architecture document for the telar monorepo consolidation*

## 🏗️ **Updated Platform Architecture**

### **Overview**
The `telar` monorepo consolidates telar-core, telar-web, and telar-social-go into a unified, professional social platform with vertical slice architecture and AI-powered features. The platform includes a dedicated AI Engine microservice for content moderation, analysis, and intelligent content processing.

## 📁 **Updated Directory Structure**

```
telar/
├── apps/                      # 🎯 Deployable applications
│   ├── api/                   # Go backend with vertical slice architecture
│   │   ├── cmd/               # Application entry points
│   │   │   ├── server/        # Main API server entry point
│   │   │   │   └── bootstrap/ # Server bootstrap logic
│   │   │   └── services/      # Standalone service entry points
│   │   │       ├── auth/     # Auth service (Dockerfile, main.go)
│   │   │       ├── comments/  # Comments service (Dockerfile, main.go)
│   │   │       ├── posts/     # Posts service (Dockerfile, main.go)
│   │   │       └── profile/   # Profile service (Dockerfile, main.go)
│   │   ├── internal/          # Private Go code (migrated from telar-core)
│   │   │   ├── cache/         # Caching layer (Redis, memory, cluster)
│   │   │   ├── config/        # Configuration management
│   │   │   ├── database/      # Repository layer (enhanced)
│   │   │   │   ├── factory/   # Repository factory pattern
│   │   │   │   ├── interfaces/ # Repository interfaces
│   │   │   │   ├── observability/ # Database metrics
│   │   │   │   ├── postgres/  # PostgreSQL client
│   │   │   │   ├── postgresql/ # PostgreSQL repository implementation
│   │   │   │   └── utils/     # Database utilities
│   │   │   ├── middleware/    # Authentication and request middleware
│   │   │   │   ├── admin/     # Admin authorization
│   │   │   │   ├── authhmac/  # HMAC authentication
│   │   │   │   ├── authjwt/   # JWT authentication
│   │   │   │   ├── authrole/  # Role-based authorization
│   │   │   │   ├── constraints/ # Request constraints
│   │   │   │   ├── dualauth/  # Dual authentication
│   │   │   │   ├── ratelimit/ # Rate limiting
│   │   │   │   └── requestid/ # Request ID tracking
│   │   │   ├── platform/      # Base service and utilities
│   │   │   │   ├── config/    # Platform configuration
│   │   │   │   └── email/     # Email sending (SMTP)
│   │   │   ├── pkg/           # Package utilities
│   │   │   │   ├── content/   # Content processing
│   │   │   │   ├── env/       # Environment variable processing
│   │   │   │   ├── log/       # Logging utilities
│   │   │   │   └── parser/    # Parsing utilities
│   │   │   ├── recaptcha/     # reCAPTCHA verification
│   │   │   ├── server/        # Server utilities
│   │   │   ├── testutil/      # Testing utilities and helpers
│   │   │   ├── types/         # Type definitions
│   │   │   └── utils/         # Utility functions
│   │   ├── auth/              # Use case vertical slice (2,838 lines)
│   │   ├── posts/             # Use case vertical slice (1,328 lines)
│   │   ├── profile/           # Use case vertical slice (1,000 lines)
│   │   ├── comments/          # Simple vertical slice (765 lines)
│   │   ├── notifications/     # Simple vertical slice (826 lines)
│   │   ├── user-rels/         # Simple vertical slice (694 lines)
│   │   ├── votes/             # Simple vertical slice (636 lines)
│   │   ├── setting/           # Simple vertical slice (627 lines)
│   │   ├── gallery/           # Simple vertical slice (603 lines)
│   │   ├── circles/           # Simple vertical slice (451 lines)
│   │   ├── admin/             # Simple vertical slice (476 lines)
│   │   ├── actions/           # Simple vertical slice (476 lines)
│   │   ├── storage/           # Simple vertical slice (281 lines)
│   │   ├── bookmarks/         # Bookmarks vertical slice
│   │   ├── orchestrator/      # Service orchestration layer
│   │   │   └── signup/        # Signup orchestration
│   │   └── shared/            # Shared interfaces between services
│   │       └── interfaces/    # Service-to-service interfaces
│   ├── ai-engine/             # AI-powered content moderation microservice
│   │   ├── cmd/api/           # AI Engine API entry point
│   │   ├── internal/          # AI Engine internals
│   │   │   ├── analyzer/      # Content analysis service
│   │   │   ├── api/           # API handlers and router
│   │   │   ├── config/        # Configuration
│   │   │   ├── generator/     # Content generation
│   │   │   ├── knowledge/     # Knowledge base service
│   │   │   ├── middleware/    # Internal authentication
│   │   │   ├── moderation/   # Moderation pipeline (ONNX, LLM, keyword filters)
│   │   │   ├── platform/      # Platform integrations
│   │   │   │   ├── llm/       # LLM adapters (Groq, OpenAI, OpenRouter, Ollama)
│   │   │   │   └── weaviate/  # Weaviate vector database client
│   │   │   ├── processor/     # Content processing
│   │   │   │   └── chunker/   # Text chunking
│   │   │   └── prompt/        # Prompt registry
│   │   ├── deployments/       # Docker Compose deployment
│   │   ├── prompts/           # AI prompt templates
│   │   ├── scripts/           # Model export and comparison scripts
│   │   └── public/            # Static assets
│   └── web/                   # Unified Next.js frontend
│       ├── src/
│       │   ├── app/           # App Router with domain-based routing
│       │   │   ├── (admin)/   # Admin route group
│       │   │   ├── (auth)/    # Auth route group
│       │   │   ├── (dashboard)/ # Dashboard route group
│       │   │   └── api/       # API routes (Next.js API handlers)
│       │   ├── components/    # Shared UI components
│       │   │   ├── auth/      # Auth-specific components
│       │   │   ├── layouts/  # Layout components
│       │   │   └── ThemeRegistry/ # Theme provider
│       │   ├── features/      # Feature-based domain modules
│       │   │   ├── account/   # Account management
│       │   │   ├── admin/     # Admin features
│       │   │   ├── auth/      # Authentication features
│       │   │   ├── bookmarks/ # Bookmarks features
│       │   │   ├── comments/  # Comments features
│       │   │   ├── moderation/ # Moderation features
│       │   │   ├── posts/     # Posts features
│       │   │   ├── profile/   # Profile features
│       │   │   ├── search/    # Search features
│       │   │   ├── storage/   # Storage utilities
│       │   │   └── votes/     # Voting features
│       │   ├── lib/           # Shared utilities and libraries
│       │   │   ├── api/       # API client utilities
│       │   │   ├── auth/      # Auth utilities (cookies, JWT)
│       │   │   ├── i18n/      # Internationalization
│       │   │   ├── provider/  # React providers
│       │   │   ├── react-query/ # React Query setup
│       │   │   └── theme/     # Theme management
│       │   └── middleware.ts  # Next.js middleware
│       ├── public/
│       │   └── locales/       # Internationalization files
│       │       ├── ar/        # Arabic translations
│       │       ├── en/        # English translations
│       │       ├── es/        # Spanish translations
│       │       ├── fa/        # Farsi translations
│       │       ├── fr/        # French translations
│       │       └── zh/        # Chinese translations
│       └── tests/             # E2E tests (Playwright)
│
├── packages/                  # 🎯 Shared libraries and configurations
│   ├── sdk/                   # TypeScript client SDK
│   │   ├── src/              # SDK source code
│   │   │   ├── admin.ts      # Admin API client
│   │   │   ├── auth.ts       # Auth API client
│   │   │   ├── bookmarks.ts  # Bookmarks API client
│   │   │   ├── client.ts     # Base HTTP client
│   │   │   ├── comments.ts   # Comments API client
│   │   │   ├── config.ts     # SDK configuration
│   │   │   ├── posts.ts      # Posts API client
│   │   │   ├── profile.ts    # Profile API client
│   │   │   ├── storage.ts    # Storage API client
│   │   │   ├── storage-utils.ts # Storage utilities
│   │   │   ├── types.ts      # TypeScript type definitions
│   │   │   └── votes.ts      # Votes API client
│   │   └── eslint.config.mjs # ESLint configuration
│   ├── clients/               # Go client libraries
│   │   └── aiengine/          # AI Engine Go client
│   │       ├── client.go     # AI Engine client implementation
│   │       └── go.mod        # Go module
│   └── config-eslint/         # Shared ESLint configuration
│
├── deployments/               # 🎯 Platform-specific deployments
│   ├── docker-compose/        # Docker Compose for development
│   │   ├── docker-compose.yml
│   │   ├── docker-compose.prod.yml
│   │   └── .env.example
│   └── [ai-engine]/           # AI Engine deployments (in apps/ai-engine/deployments/)
│       └── docker-compose/    # AI Engine Docker Compose setup
│   ├── kubernetes/            # Kubernetes production deployment
│   │   ├── manifests/
│   │   │   ├── namespace.yaml
│   │   │   ├── configmap.yaml
│   │   │   ├── secrets.yaml
│   │   │   ├── services/
│   │   │   ├── deployments/
│   │   │   └── ingress/
│   │   └── helm/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       └── templates/
│   ├── encore/                # Encore.dev serverless platform
│   │   ├── encore.app
│   │   ├── services/
│   │   │   ├── auth/
│   │   │   ├── posts/
│   │   │   └── comments/
│   │   └── config/
│   ├── openfaas/              # OpenFaaS serverless platform
│   │   ├── stack.yml
│   │   ├── config/
│   │   └── functions/
│   └── aws-lambda/            # AWS Lambda serverless
│       ├── serverless.yml
│       ├── adapters/
│       └── functions/
│
├── platforms/                 # 🎯 Platform adapters and abstractions
│   ├── interfaces/            # Platform abstraction interfaces
│   │   ├── platform.go        # Main platform interface
│   │   ├── deployment.go      # Deployment interface
│   │   └── scaling.go         # Scaling interface
│   ├── docker/                # Docker platform adapter
│   │   ├── adapter.go
│   │   └── compose.go
│   ├── kubernetes/            # Kubernetes platform adapter
│   │   ├── adapter.go
│   │   └── manifests.go
│   ├── encore/                # Encore.dev platform adapter
│   │   ├── adapter.go
│   │   └── services.go
│   ├── openfaas/              # OpenFaaS platform adapter
│   │   ├── adapter.go
│   │   └── functions.go
│   └── serverless/            # Generic serverless adapter
│       ├── adapter.go
│       └── aws.go
│
├── dashboard/                 # 🎯 Admin dashboard and management
│   ├── web/                   # Web-based admin interface
│   │   ├── src/
│   │   ├── public/
│   │   └── package.json
│   ├── api/                   # Dashboard API
│   │   ├── handlers/
│   │   ├── models/
│   │   └── services/
│   └── config/                # Dashboard configuration
│
├── marketplace/               # 🎯 AI marketplace and plugins
│   ├── plugins/               # AI plugin system
│   │   ├── conversation-starters/
│   │   ├── content-summarization/
│   │   ├── smart-moderation/
│   │   └── personalized-feeds/
│   ├── api/                   # Marketplace API
│   │   ├── handlers/
│   │   ├── models/
│   │   └── services/
│   └── web/                   # Marketplace web interface
│       ├── src/
│       ├── public/
│       └── package.json
│
├── docs/                      # 🎯 Comprehensive documentation
│   ├── api-reference/         # OpenAPI/Swagger specifications
│   │   ├── actions.yaml       # Actions API spec
│   │   ├── admin.yaml         # Admin API spec
│   │   ├── auth.yaml          # Auth API spec
│   │   ├── circles.yaml       # Circles API spec
│   │   ├── comments.yaml      # Comments API spec
│   │   ├── common.yaml        # Common API definitions
│   │   ├── gallery.yaml       # Gallery API spec
│   │   ├── notifications.yaml # Notifications API spec
│   │   ├── posts.yaml         # Posts API spec
│   │   ├── profile.yaml       # Profile API spec
│   │   ├── setting.yaml       # Settings API spec
│   │   ├── storage.yaml       # Storage API spec
│   │   ├── userrels.yaml      # User relations API spec
│   │   └── votes.yaml         # Votes API spec
│   ├── architecture/          # Architecture documentation
│   │   ├── platform_architecture_structure.md # This file
│   │   ├── professional_architecture_blueprint.md # Blueprint
│   │   └── go-workspaces.md   # Go workspaces guide
│   ├── development/           # Development guides
│   │   ├── AUTH_MICROSERVICE_COMPLETE_ANALYSIS.md # Auth analysis
│   │   └── linting.md         # Linting guide
│   ├── notes/                 # Development notes
│   │   ├── backend/           # Backend notes
│   │   └── architectural-notes/ # Architecture notes
│   ├── deployment/            # Deployment guides (planned)
│   │   ├── docker.md
│   │   ├── kubernetes.md
│   │   ├── encore.md
│   │   └── openfaas.md
│   └── diagrams/              # Architecture diagrams (planned)
│       ├── platform-architecture.svg
│       ├── service-flow.svg
│       └── deployment-options.svg
│
├── protos/                    # 🎯 Protocol Buffer definitions (gRPC)
│   ├── comments/              # Comments service protobuf
│   │   └── v1/
│   │       └── comments.proto
│   ├── posts/                 # Posts service protobuf
│   │   └── v1/
│   │       └── posts.proto
│   ├── profile/               # Profile service protobuf
│   │   └── v1/
│   │       └── profile.proto
│   └── gen/                   # Generated code
│       └── go/                # Generated Go code
│           ├── comments/      # Comments generated code
│           ├── commentspb/   # Comments package
│           ├── posts/         # Posts generated code
│           ├── postspb/       # Posts package
│           ├── profile/       # Profile generated code
│           └── profilepb/     # Profile package
│
├── tools/                     # 🎯 Development and deployment tools
│   ├── deploy/                # Deployment tools (planned)
│   │   ├── deploy.go
│   │   ├── platform-cli/
│   │   └── scripts/
│   ├── migrate/               # Migration tools (planned)
│   │   ├── migrate.go
│   │   └── scripts/
│   ├── dev/                   # Development tools
│   │   ├── app/               # Application management scripts
│   │   │   ├── start.sh       # Start servers
│   │   │   ├── start-api-only.sh # Start API only
│   │   │   ├── start-ai-engine.sh # Start AI Engine
│   │   │   ├── restart.sh    # Restart servers
│   │   │   └── stop.sh        # Stop servers
│   │   ├── infra/             # Infrastructure scripts
│   │   │   ├── check-env.sh   # Environment checks
│   │   │   ├── db-migrate.sh  # Database migrations
│   │   │   └── start-docker.sh # Docker startup
│   │   ├── lib/               # Shared library scripts
│   │   │   └── common.sh      # Common utilities
│   │   ├── seed/              # Test data seeding scripts
│   │   │   ├── all.sh         # Seed all data
│   │   │   ├── users.sh       # Seed users
│   │   │   ├── posts.sh       # Seed posts
│   │   │   ├── comments.sh    # Seed comments
│   │   │   └── bookmarks.sh   # Seed bookmarks
│   │   ├── test/              # E2E test scripts
│   │   │   ├── e2e-auth.sh    # Auth E2E tests
│   │   │   ├── e2e-posts.sh   # Posts E2E tests
│   │   │   ├── e2e-comments.sh # Comments E2E tests
│   │   │   ├── e2e-profile.sh # Profile E2E tests
│   │   │   ├── e2e-votes.sh   # Votes E2E tests
│   │   │   ├── lifecycle-main.sh # Lifecycle tests
│   │   │   ├── stress-workflow.sh # Stress tests
│   │   │   └── verify-release.sh # Release verification
│   │   └── test_env.sh        # Test environment setup
│   ├── linters/               # Custom linting tools
│   │   ├── main.go            # Linter entry point
│   │   ├── no_setenv_in_tests.go # Test linter rules
│   │   └── README.md          # Linter documentation
│   └── protoc/                # Protocol Buffer compiler tools
│       ├── include/           # Protobuf includes
│       │   └── google/        # Google protobuf definitions
│       └── readme.txt         # Usage instructions
│
├── examples/                  # 🎯 Example implementations
│   ├── basic-social/          # Basic social network setup
│   │   ├── docker-compose.yml
│   │   ├── config/
│   │   └── README.md
│   ├── community-forum/       # Community forum setup
│   │   ├── docker-compose.yml
│   │   ├── config/
│   │   └── README.md
│   └── enterprise-social/     # Enterprise social network
│       ├── kubernetes/
│       ├── config/
│       └── README.md
│
├── config/                    # 🎯 Global configuration
│   ├── app_config.yml         # Main application configuration
│   ├── database_config.yml    # Database configuration
│   ├── ai_config.yml          # AI features configuration
│   └── env/                   # Environment-specific configs
│       ├── development.yml
│       ├── staging.yml
│       └── production.yml
│
├── constants/                 # 🎯 Application constants
│   ├── auth_keywords_const.go
│   ├── feed_const.go
│   ├── album_const.go
│   └── other_constants.go
│
├── swag/                      # 🎯 API documentation generation
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
│
├── docker/                    # 🎯 Docker configurations
│   ├── Dockerfile
│   ├── Dockerfile.dev
│   └── docker-compose.yml
│
├── scripts/                   # 🎯 Utility scripts
│   ├── setup.sh
│   ├── deploy.sh
│   ├── migrate.sh
│   └── test.sh
│
├── go.mod                     # Go module definition
├── go.sum                     # Go dependencies
├── README.md                  # Main project documentation
├── LICENSE                    # MIT License
└── .gitignore                 # Git ignore rules
```

## 🎯 **Key Architectural Components**

### **1. Core Layer (`core/`)**
- **Purpose**: Shared infrastructure and utilities
- **Source**: Consolidated from `telar-core`
- **Features**: Data repositories, middleware, utilities, types

### **2. Services Layer (`apps/api/`)**
- **Purpose**: Microservices for all social features
- **Source**: Consolidated from `telar-web` and `telar-social-go`
- **Features**: Auth, profiles, posts, comments, circles, gallery, votes, bookmarks, storage, admin, notifications, user-rels, setting, actions
- **Architecture**: Vertical slice architecture with use case slices and simple slices
- **Communication**: gRPC for inter-service communication (protobuf definitions in `protos/`)
- **Orchestration**: `orchestrator/` layer for complex multi-service workflows
- **Shared Interfaces**: `shared/interfaces/` for service-to-service contracts

### **3. Deployment Layer (`deployments/`)**
- **Purpose**: Platform-specific deployment configurations
- **Features**: Docker, Kubernetes, Encore.dev, OpenFaaS, AWS Lambda

### **4. Platform Layer (`platforms/`)**
- **Purpose**: Platform abstraction and adapters
- **Features**: Unified interface for different deployment platforms

### **5. Dashboard Layer (`dashboard/`)**
- **Purpose**: Admin interface and management tools
- **Features**: Web-based admin, analytics, user management

### **6. Marketplace Layer (`marketplace/`)**
- **Purpose**: AI plugins and marketplace
- **Features**: AI features, plugin system, marketplace API

### **7. AI Engine Layer (`apps/ai-engine/`)**
- **Purpose**: AI-powered content moderation and analysis microservice
- **Features**: 
  - Content moderation pipeline (ONNX models, LLM adapters, keyword filters)
  - Content analysis and classification
  - Vector search integration (Weaviate)
  - Multiple LLM provider support (Groq, OpenAI, OpenRouter, Ollama)
  - Knowledge base service
  - Content generation
- **Deployment**: Docker Compose with model preloading
- **Models**: ONNX-based toxicity and spam detection models

### **8. Protocol Buffers Layer (`protos/`)**
- **Purpose**: gRPC service definitions and generated code
- **Services**: Comments, Posts, Profile
- **Generated Code**: Go packages for gRPC clients and servers
- **Location**: `protos/gen/go/` contains generated code

### **9. Client Libraries Layer (`packages/clients/`)**
- **Purpose**: Go client libraries for internal service communication
- **Clients**: AI Engine client for Go services to interact with AI Engine
- **Usage**: Used by API services to call AI Engine for content moderation

### **10. Development Tools Layer (`tools/dev/`)**
- **Purpose**: Comprehensive development and testing tooling
- **Components**:
  - **app/**: Server lifecycle management (start, stop, restart)
  - **infra/**: Infrastructure setup (Docker, databases, migrations)
  - **seed/**: Test data generation for development and testing
  - **test/**: E2E test orchestration scripts
  - **lib/**: Shared shell utilities
- **Linters**: Custom Go linting rules (`tools/linters/`)
- **Protoc**: Protocol buffer compiler setup (`tools/protoc/`)

## 🔄 **Updated Migration Strategy**

### **Phase 1: Core Infrastructure Migration (Day 2)**
```
telar-core/ → apps/api/internal/
├── config/ → internal/config/
├── middleware/ → internal/middleware/
├── utils/ → internal/utils/
├── types/ → internal/types/
├── server/ → internal/server/
├── pkg/ → internal/pkg/
└── data/ → internal/database/mongodb/ (enhanced)
```

### **Phase 2: Repository Layer Enhancement (Day 3)**
```
telar-social-go/pkg/repository/ → apps/api/internal/database/
├── interfaces/ → internal/database/interfaces/
├── factory/ → internal/database/factory/
├── mongodb/ → internal/database/mongodb/ (merged)
└── postgresql/ → internal/database/postgresql/

telar-social-go/pkg/service/ → apps/api/internal/platform/
└── base_service.go → internal/platform/base_service.go
```

### **Phase 3: Vertical Slice Migration (Day 3-4)**
```
telar-web/micros/ → apps/api/
├── auth/ → auth/ (use case vertical slice)
├── profile/ → profile/ (use case vertical slice)
├── notifications/ → notifications/ (simple vertical slice)
├── setting/ → setting/ (simple vertical slice)
├── admin/ → admin/ (simple vertical slice)
├── actions/ → actions/ (simple vertical slice)
└── storage/ → storage/ (simple vertical slice)

telar-social-go/micros/ → apps/api/
├── posts/ → posts/ (use case vertical slice)
├── comments/ → comments/ (simple vertical slice)
├── circles/ → circles/ (simple vertical slice)
├── gallery/ → gallery/ (simple vertical slice)
├── votes/ → votes/ (simple vertical slice)
└── user-rels/ → user-rels/ (simple vertical slice)
```

### **Phase 4: Frontend Integration (Day 5-6)**
```
telar-web/frontend/ → apps/web/
├── src/ → src/
├── public/ → public/
└── package.json → package.json
```

### **Phase 5: AI Engine Integration (Post-Migration)**
```
New AI Engine microservice:
├── apps/ai-engine/            # Standalone AI moderation service
├── packages/clients/aiengine/ # Go client for AI Engine
└── Integration with posts/    # Content moderation pipeline
```

### **Phase 6: gRPC Service Communication (Post-Migration)**
```
Protocol Buffer definitions:
├── protos/                    # Service definitions
├── protos/gen/go/            # Generated Go code
└── Integration with services/ # gRPC client/server setup
```

## 🚀 **Deployment Options**

### **1. Docker Compose (Development)**
```bash
cd social-ai-platform
docker-compose up -d
```

### **2. Kubernetes (Production)**
```bash
kubectl apply -f deployments/kubernetes/manifests/
```

### **3. Encore.dev (Serverless)**
```bash
cd social-ai-platform/deployments/encore
encore run
```

### **4. OpenFaaS (Serverless)**
```bash
cd social-ai-platform/deployments/openfaas
faas-cli deploy -f stack.yml
```

### **5. AWS Lambda (Serverless)**
```bash
cd social-ai-platform/deployments/aws-lambda
serverless deploy
```
