FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ ./

RUN CGO_ENABLED=0 go build -o /api-server ./cmd/services/auth/main.go

FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /home/appuser

COPY --from=builder /api-server .

CMD ["./api-server"]
