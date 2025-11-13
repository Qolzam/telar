# Posts Service

Posts microservice for the Telar social network platform. Manages user posts including content, media, comments, voting, and sharing capabilities.

## Port

- **8082**: Posts service HTTP endpoint

## API Endpoints

### User-Facing Routes (Dual Auth - JWT/Cookie)
- `POST /posts` - Create a new post
- `PUT /posts` - Update a post
- `PUT /posts/profile` - Update post profile information
- `PUT /posts/comment/disable` - Disable comments on a post
- `PUT /posts/share/disable` - Disable sharing on a post
- `PUT /posts/urlkey/:postId` - Generate URL key for a post
- `DELETE /posts/:postId` - Delete a post
- `GET /posts` - Query posts with filters
- `GET /posts/cursor` - Query posts with cursor-based pagination
- `GET /posts/cursor/:postId` - Get cursor info for a post
- `GET /posts/search/cursor` - Search posts with cursor-based pagination
- `GET /posts/:postId` - Get post by ID
- `GET /posts/urlkey/:urlkey` - Get post by URL key

### Service-to-Service Routes (HMAC Auth)
- `POST /posts/index` - Create database indexes
- `PUT /posts/score` - Increment post score
- `PUT /posts/comment/count` - Increment comment count

## Setup

### Prerequisites
- PostgreSQL database
- Environment variables configured (see `.env`)

### Running Locally
```bash
# Start PostgreSQL
make up-postgres

# Run posts service
make run-posts
```

### Running with Docker
```bash
# Build and run with docker-compose
docker-compose up posts
```

## Environment Variables

The service uses the same environment configuration as other Telar microservices:
- `DB_TYPE` - Database type (`postgresql`)
- `POSTGRES_HOST` - PostgreSQL connection string
- `JWT_PUBLIC_KEY` - JWT public key for token verification
- `HMAC_SECRET` - HMAC secret for service-to-service auth
- See `apps/api/.env` for complete configuration

## Development

### Testing
```bash
# Run unit tests
make test-posts

# Run e2e tests
bash tools/dev/scripts/posts_e2e_test.sh
```

### Building
```bash
# Build binary
cd apps/api
go build -o posts-service ./cmd/services/posts/main.go

# Build Docker image
docker build -f cmd/services/posts/Dockerfile -t telar-posts:latest .
```

## Architecture

The posts service follows clean architecture principles:
- **Handler Layer**: HTTP request handling and routing
- **Service Layer**: Business logic and data transformation
- **Repository Layer**: Database abstraction (via BaseService)
- **Validation Layer**: Request validation and sanitization
- **Error Handling**: Standardized error responses

## Dependencies

- **Fiber v2**: Web framework
- **PostgreSQL**: Primary database backend
- **Platform**: Shared platform services and utilities

