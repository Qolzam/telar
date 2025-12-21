# Auth Microservice

This service handles all authentication, authorization, and user management logic.

## Running Locally

1. **Ensure Docker is running.**

2. **Create the master configuration file.** From the root of the project, navigate to the `apps/api` directory and create your local `.env` file if it doesn't exist:
   ```bash
   cd apps/api
   cp .env.example .env
   cd ../..
   ```

3. **Start the service and its dependencies.** From this directory (`apps/api/cmd/services/auth`), run:
   ```bash
   docker-compose up --build
   ```

The service will be available at `http://localhost:9099`.

## Configuration

The service is configured via environment variables. These are loaded from the central `apps/api/.env` file by the `docker-compose.yml`. See `apps/api/.env.example` for a complete list of required variables.

## Features

- **JWT-based Authentication**: Secure token-based authentication with configurable expiry
- **HMAC Service-to-Service Auth**: Secure inter-service communication
- **Password Management**: Secure password hashing, reset, and change functionality
- **Email Verification**: Email-based user verification with rate limiting
- **OAuth Integration**: Support for GitHub and Google OAuth
- **Rate Limiting**: Configurable rate limiting for all endpoints
- **Security Features**: Input validation, SQL injection protection, XSS prevention
- **Database Support**: PostgreSQL
- **Monitoring**: Health checks, metrics, and structured logging

## API Endpoints

### Authentication
- `POST /auth/signup` - User registration
- `POST /auth/login` - User login
- `GET /auth/login/github` - GitHub OAuth redirect
- `GET /auth/login/google` - Google OAuth redirect
- `GET /auth/oauth2/authorized` - OAuth callback

### Password Management
- `POST /auth/password/forget` - Request password reset
- `GET /auth/password/forget` - Password forget page
- `POST /auth/password/reset/:verifyId` - Reset password
- `GET /auth/password/reset/:verifyId` - Password reset page
- `PUT /auth/password/change` - Change password (requires JWT)

### Verification
- `POST /auth/signup/verify` - Verify email/phone code

### Admin
- `POST /auth/admin/check` - Check admin status (HMAC required)
- `POST /auth/admin/signup` - Admin registration (HMAC required)
- `POST /auth/admin/login` - Admin login (HMAC required)

### Profile
- `PUT /auth/profile` - Update user profile (requires JWT)

### System
- `GET /auth/.well-known/jwks.json` - JWKS endpoint

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with database tests
RUN_DB_TESTS=1 go test ./...

# Run with race detection
go test -race ./...
```

### Building

```bash
# Build for current platform
go build -o auth-service ./apps/api/cmd/services/auth/

# Build for Linux
GOOS=linux go build -o auth-service ./apps/api/cmd/services/auth/

# Build with optimizations
go build -ldflags="-s -w" -o auth-service ./apps/api/cmd/services/auth/
```

## Production Deployment

In production, environment variables are provided by your orchestration platform (Kubernetes secrets, AWS Parameter Store, etc.), not from `.env` files. The Docker image is environment-agnostic and expects configuration to be injected at runtime.

## License

This project is part of the Telar platform and follows the same licensing terms.
