# Comments Service

Comments microservice for the Telar social network platform. Handles threaded discussions, profile updates, and moderation helpers for posts.

## Port

- **8083**: Comments service HTTP endpoint

## API Endpoints

All routes support dual authentication (JWT/Cookie with HMAC fallback):

- `POST /comments` - Create a new comment
- `PUT /comments` - Update an existing comment
- `GET /comments?postId=` - Query comments for a post with pagination
- `PUT /comments/profile` - Bulk update commenter profile details
- `GET /comments/:commentId` - Retrieve a single comment
- `DELETE /comments/id/:commentId/post/:postId` - Delete a specific comment
- `DELETE /comments/post/:postId` - Delete all comments for a post

## Setup

### Prerequisites
- PostgreSQL database
- Environment variables configured (see `apps/api/.env`)

### Running Locally
```bash
# Start PostgreSQL
make up-postgres

# Run comments service
make run-comments
```

### Running with Docker
```bash
# Build and run with docker-compose
docker compose up comments
```

## Environment Variables

The service uses the shared platform configuration:
- `DB_TYPE` - Database type (`postgresql`)
- `POSTGRES_HOST` - PostgreSQL connection string
- `JWT_PUBLIC_KEY` - JWT public key for token verification
- `HMAC_SECRET` - HMAC secret for service-to-service auth
- See `apps/api/.env` for complete configuration

## Development

### Testing
```bash
# Run unit and integration tests
make test-comments

# Run E2E workflow
make test-e2e-comments
```

### Building
```bash
# Build binary
cd apps/api
go build -o comments-service ./cmd/services/comments/main.go

# Build Docker image
docker build -f cmd/services/comments/Dockerfile -t telar-comments:latest .
```

## Architecture

The comments service mirrors the posts vertical slice:
- **Handler Layer**: Fiber handlers with dual-auth middleware
- **Service Layer**: Business logic, caching, and repository operations
- **Repository Layer**: Database abstraction via `BaseService`
- **Validation Layer**: Request validation and sanitization
- **Error Handling**: Standardized error responses with contextual messages

## Dependencies

- **Fiber v2**: Web framework
- **PostgreSQL**: Primary database backend
- **Platform**: Shared platform services and utilities

