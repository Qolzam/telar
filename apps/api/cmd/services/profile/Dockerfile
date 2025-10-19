# Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ ./

RUN CGO_ENABLED=0 go build -o /profile-service ./cmd/services/profile/main.go

# Stage 2: Final Production Image
FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /home/appuser

COPY --from=builder /profile-service .

CMD ["./profile-service"]







