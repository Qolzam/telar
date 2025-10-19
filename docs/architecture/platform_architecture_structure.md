# Telar Platform Architecture & Structure
*Updated architecture document for the telar monorepo consolidation*

## 🏗️ **Updated Platform Architecture**

### **Overview**
The `telar` monorepo consolidates telar-core, telar-web, and telar-social-go into a unified, professional social platform with vertical slice architecture and AI-powered features.

## 📁 **Updated Directory Structure**

```
telar/
├── apps/                      # 🎯 Deployable applications
│   ├── api/                   # Go backend with vertical slice architecture
│   │   ├── cmd/server/        # Application entry point
│   │   ├── internal/          # Private Go code (migrated from telar-core)
│   │   │   ├── config/        # Configuration management
│   │   │   ├── database/      # Repository layer (enhanced)
│   │   │   ├── middleware/    # Authentication middleware
│   │   │   ├── platform/      # Base service and utilities
│   │   │   ├── pkg/           # Package utilities
│   │   │   ├── server/        # Server utilities
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
│   │   └── storage/           # Simple vertical slice (281 lines)
│   └── web/                   # Unified Next.js frontend
│       ├── src/app/           # App Router with domain-based routing
│       ├── src/components/    # Shared components
│       ├── src/lib/           # Shared utilities
│       └── src/hooks/         # Shared hooks
│
├── packages/                  # 🎯 Shared libraries and configurations
│   ├── sdk/                   # TypeScript client SDK
│   │   ├── src/              # SDK source code
│   │   ├── api/              # API client functions
│   │   ├── types/            # TypeScript type definitions
│   │   └── utils/            # SDK utilities
│   └── config-eslint/         # Shared ESLint configuration
│
├── deployments/               # 🎯 Platform-specific deployments
│   ├── docker-compose/        # Docker Compose for development
│   │   ├── docker-compose.yml
│   │   ├── docker-compose.prod.yml
│   │   └── .env.example
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
│   ├── deployment/            # Deployment guides
│   │   ├── docker.md
│   │   ├── kubernetes.md
│   │   ├── encore.md
│   │   └── openfaas.md
│   ├── development/           # Development guides
│   │   ├── setup.md
│   │   ├── contributing.md
│   │   └── architecture.md
│   ├── api/                   # API documentation
│   │   ├── auth.md
│   │   ├── posts.md
│   │   ├── comments.md
│   │   └── swagger/
│   └── diagrams/              # Architecture diagrams
│       ├── platform-architecture.svg
│       ├── service-flow.svg
│       └── deployment-options.svg
│
├── tools/                     # 🎯 Development and deployment tools
│   ├── deploy/                # Deployment tools
│   │   ├── deploy.go
│   │   ├── platform-cli/
│   │   └── scripts/
│   ├── migrate/               # Migration tools
│   │   ├── migrate.go
│   │   └── scripts/
│   └── dev/                   # Development tools
│       ├── dev.go
│       └── scripts/
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

### **2. Services Layer (`services/`)**
- **Purpose**: Microservices for all social features
- **Source**: Consolidated from `telar-web` and `telar-social-go`
- **Features**: Auth, profiles, posts, comments, circles, gallery, votes

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
