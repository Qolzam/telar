# Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy modules and download dependencies
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

# Copy the entire API source
COPY apps/api/ ./

# Build the auth service binary
RUN CGO_ENABLED=0 go build -o /auth-service ./cmd/services/auth/main.go

# Stage 2: Final Production Image
FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /home/appuser

COPY --from=builder /auth-service .

CMD ["./auth-service"]
