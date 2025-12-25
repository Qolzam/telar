# Go Workspaces: Monorepo Dependency Management

## Overview

This monorepo uses **Go Workspaces** (introduced in Go 1.18+) to manage dependencies across multiple modules. This is the modern, idiomatic approach for Go monorepos, replacing the need for `replace` directives in `go.mod` files.

## Structure

The repository root contains a `go.work` file that defines the workspace:

```
go.work
├── ./apps/api
├── ./apps/ai-engine
├── ./packages/clients/aiengine
└── ./tools/linters
```

## Why Go Workspaces?

1. **Clean Dependency Management**: No need for `replace` directives in individual `go.mod` files
2. **IDE Support**: Modern IDEs (VS Code, GoLand) automatically recognize workspace structure
3. **Type Safety**: Direct imports between local modules work seamlessly
4. **Versioning Flexibility**: Each module maintains its own `go.mod` but can reference local packages directly

## Module Strategy

### Local Modules (Workspace-Only)

Modules like `packages/clients/aiengine` are designed to be consumed only within the workspace:

- **Purpose**: Shared packages used by multiple services
- **Versioning**: Not versioned independently (uses workspace version)
- **Publishing**: Not published to external repositories
- **Consumption**: Imported directly: `github.com/qolzam/telar/packages/clients/aiengine`

### Service Modules

Service modules (`apps/api`, `apps/ai-engine`) are full-fledged Go applications:

- **Purpose**: Standalone microservices
- **Versioning**: May be versioned independently for deployment
- **Dependencies**: Can use both external packages and workspace-local packages

## Development Workflow

### Adding a New Module to the Workspace

```bash
# 1. Create the module
cd packages/new-package
go mod init github.com/qolzam/telar/packages/new-package

# 2. Add to workspace
cd ../..
go work use ./packages/new-package
```

### Building

```bash
# Build all workspace modules
go build ./...

# Build specific module
go build ./apps/api/...

# Build from within a module
cd apps/api
go build ./...
```

### Running Tests

```bash
# Test all modules
go test ./...

# Test specific module
go test ./apps/api/...

# Test from within module
cd apps/api
go test ./...
```

## CI/CD Considerations

1. **Commit `go.work`**: The `go.work` file should be committed to version control
2. **No Extra Steps**: Go Workspaces work seamlessly in CI - no special setup required
3. **Build Commands**: Use standard `go build` and `go test` commands
4. **Cache Efficiency**: Go's module cache works efficiently with workspaces

## Best Practices

1. **Module Boundaries**: Each module should have a clear, single responsibility
2. **Shared Packages**: Place shared code in `packages/` directory
3. **Service Isolation**: Services in `apps/` should be independently deployable
4. **Dependency Direction**: 
   - Services can depend on packages
   - Packages should not depend on services
   - Avoid circular dependencies

## Troubleshooting

### "missing metadata" or "cannot find package" errors

- Ensure the module is added to `go.work`: `go work use ./path/to/module`
- Verify the import path matches the module name in `go.mod`
- Run `go work sync` to refresh workspace state

### IDE not recognizing workspace

- Restart the IDE/editor
- Run `go work sync` in the repository root
- Ensure you're opening the repository root (not a subdirectory)

## References

- [Go Workspaces Proposal](https://go.dev/ref/mod#workspaces)
- [Go Workspaces Tutorial](https://go.dev/doc/tutorial/workspaces)

