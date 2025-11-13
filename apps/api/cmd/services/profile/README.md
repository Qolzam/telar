# Profile Service

Profile microservice for the Telar social network platform. Manages user profile information including personal details, social links, and profile statistics.

## Port

- **8081**: Profile service HTTP endpoint

## API Endpoints

### User-Facing Routes (Dual Auth - JWT/Cookie)
- `GET /profile/my` - Read current user's profile
- `GET /profile?search=&page=&limit=` - Query profiles with search
- `GET /profile/id/:userId` - Read profile by ID
- `GET /profile/social/:name` - Get profile by social name
- `POST /profile/ids` - Get profiles by IDs (array of UUIDs)
- `PUT /profile` - Update profile

### Service-to-Service Routes (HMAC Auth)
- `POST /profile/index` - Initialize profile indexes
- `PUT /profile/last-seen` - Update last seen timestamp
- `GET /profile/dto/id/:userId` - Read DTO profile
- `POST /profile/dto` - Create DTO profile
- `POST /profile/dispatch` - Dispatch profiles
- `POST /profile/dto/ids` - Get DTO profiles by IDs
- `PUT /profile/follow/inc/:inc/:userId` - Increase follow count
- `PUT /profile/follower/inc/:inc/:userId` - Increase follower count

## Setup

### Prerequisites
- PostgreSQL database
- Environment variables configured (see `.env`)

### Running Locally
```bash
# Start PostgreSQL
make up-postgres

# Run profile service
make run-profile
```

### Running with Docker
```bash
# Build and run with docker-compose
docker-compose up profile
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
make test-profile

# Run e2e tests
bash tools/dev/scripts/profile_e2e_test.sh
```

### Building
```bash
# Build binary
cd apps/api
go build -o profile-service ./cmd/services/profile/main.go

# Build Docker image
docker build -f cmd/services/profile/Dockerfile -t telar-profile:latest .
```

## Architecture

The profile service follows clean architecture principles:
- **Handler Layer**: HTTP request handling and routing
- **Service Layer**: Business logic and data transformation
- **Repository Layer**: Database abstraction (via BaseService)
- **Validation Layer**: Request validation and sanitization
- **Error Handling**: Standardized error responses

## Dependencies

- **Fiber v2**: Web framework
- **PostgreSQL**: Primary database backend
- **Platform**: Shared platform services and utilities










