# Telar Professional Architecture Blueprint
*Official Architecture Document - Version 2.0*

## 🎯 **Architectural Vision & Core Principles**

This architecture is designed to achieve maximum development velocity for a solo founder and maximum scalability for a future team. It is built on three core principles:

1. **Clarity over Brevity**: The structure is optimized for easy navigation and understanding. We prefer clear, explicit file and folder names.
2. **Group by Feature, Not by Layer**: All code related to a single business feature (e.g., "Posts") lives in one place. This drastically reduces the mental overhead of development and debugging.
3. **Spec-First API Development**: The API specification (OpenAPI) is the single source of truth. We generate code from the spec, we do not write documentation in code comments. This eliminates code bloat and ensures the documentation is always accurate.

## 🏗️ **High-Level Project Structure**

```
telar/
├── apps/               # Deployable applications (your products).
├── packages/           # Shared libraries and configurations.
├── deployments/         # Docker, Kubernetes, and Helm configurations.
├── docs/               # All project and API documentation.
├── tools/              # Development scripts and migration tools.
├── examples/           # Example implementations for users.
│
├── .editorconfig       # Ensures consistent coding styles across all editors.
├── .gitignore          # Specifies files and folders to be ignored by Git.
├── go.work             # Go Workspace file for the Go backend.
├── LICENSE             # Your open-source license (e.g., MIT).
├── package.json        # The ROOT package.json for managing the entire JS/TS workspace.
└── README.md           # The front door and main entry point for your project.
```

## 📁 **Detailed Breakdown: `apps/` Directory**

### **API Application (`apps/api/`)**
Your entire Go Backend, structured as a "Modular Monolith".

```
apps/api/
│
├── cmd/server/         # The main entry point for the API application.
│   └── main.go        # Initializes configs, database, router, and wires everything together.
│
├── internal/────────── # Shared Go code that is NOT importable outside of the `api` app.
│   │
│   ├── config/        # Global configuration loading (from .env, files, etc.).
│   │
│   ├── database/      # 🎯 YOUR MIGRATED `repository` LAYER.
│   │   ├── interfaces/
│   │   ├── mongodb/
│   │   ├── postgresql/
│   │   └── factory/
│   │
│   ├── platform/      # Platform-level helpers.
│   │   ├── base_service.go # 🎯 YOUR MIGRATED `service/base_service.go` lives here.
│   │   └── email/          # Example: email sending helper.
│   │
│   └── auth/────────── # 🎯 The new home for logic from your old `common.go`.
│       ├── passwords/  # Password hashing and comparison logic.
│       └── tokens/     # JWT generation and validation logic.
│
├── posts/──────────── # A "Use Case Vertical Slice" for complex features (>1000 lines).
│   ├── create/        # The "Create Post" use case.
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── model.go
│   ├── query/         # The "Query Posts" use case.
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── model.go
│   ├── update/        # The "Update Post" use case.
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── model.go
│   ├── delete/        # The "Delete Post" use case.
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── model.go
│   └── routes.go      # A function to register all post routes with the main router.
│
└── auth/──────────── # 🎯 A "Use Case Vertical Slice" for a complex feature like Auth.
    │
    ├── login/         # The "Login" use case.
    │   ├── handler.go
    │   ├── service.go
    │   └── model.go
    │
    ├── signup/        # The "Signup" use case.
    │   ├── handler.go
    │   ├── service.go
    │   └── model.go
    │
    ├── verification/  # The "Email Verification" use case.
    │   └── ...        # Follows the same handler/service/model pattern.
    │
    └── reset_password/
        └── ...
    │
    └── routes.go      # A single function to initialize all auth use cases and register their routes.
```

### **Web Application (`apps/web/`)**
Your SINGLE Unified Next.js Frontend Application.

```
apps/web/
│
├── src/
│   ├── app/           # The Next.js App Router directory.
│   │   │
│   │   ├── admin/     # Code for the `admin.telar.press` subdomain.
│   │   │   └── (dashboard)/
│   │   │
│   │   ├── app/       # Code for the core social platform at `app.telar.press`.
│   │   │   └── (main)/
│   │   │
│   │   └── marketing/ # Code for the public marketing site at `www.telar.press`.
│   │       ├── (landing)/
│   │       ├── blog/
│   │       └── ...
│   │
│   ├── components/    # Truly shared components across all parts of the Next.js app.
│   ├── lib/           # Shared library functions (e.g., date formatters).
│   └── hooks/         # Shared custom React hooks.
│
├── public/            # Static assets (images, fonts, favicons).
├── middleware.ts      # The critical file that handles domain-based routing.
└── package.json       # Dependencies for the unified web application.
```

## 📦 **Detailed Breakdown: Other Directories**

### **Packages (`packages/`)**
Reusable code packages shared across the monorepo.

```
packages/
│
├── sdk/────────────── # Your TypeScript Client SDK. Used by `apps/web` and any future app.
│   ├── src/          # The source code for the SDK.
│   └── package.json
│
└── config-eslint/──── # Shared ESLint configuration to enforce code style.
    └── index.js
```

### **Deployments (`deployments/`)**
Infrastructure-as-Code and deployment manifests.

```
deployments/
│
├── docker-compose/
│   ├── docker-compose.yml
│   └── .env.example
│
└── kubernetes/
    └── helm/
        └── telar/
```

### **Documentation (`docs/`)**
All project documentation.

```
docs/
│
├── architecture/
│   └── decisions.md   # A log of major architectural decisions (like this one).
│
├── api-reference/──── # 🎯 The home for your "Spec-First" API documentation.
│   ├── openapi.yaml   # A single master OpenAPI spec file, or...
│   ├── posts.yaml     # ...split into files per feature.
│   └── auth.yaml
│
└── guides/
    ├── quick-start.md
    └── contributing.md
```

### **Tools (`tools/`)**
Scripts for automating development tasks.

```
tools/
│
├── codegen/           # Scripts related to code generation (e.g., `oapi-codegen`).
│   └── generate.sh
│
└── migrate/           # Database migration scripts and tooling.
```

### **Examples (`examples/`)**
Complete, working examples for your users.

```
examples/
│
└── nextjs-blog-with-comments/
    ├── README.md
    └── ...
```
