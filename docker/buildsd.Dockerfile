# Build stage
FROM golang:1.23.3 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd/buildsd ./cmd/builds-server
COPY ./internal ./internal
COPY ./pkg ./pkg
COPY ./proto ./proto
COPY ./api ./api

# Build the binary
RUN go build -o /app/builds-server ./cmd/builds-server

# Final stage
FROM debian:bookworm

WORKDIR /app

COPY .env /app/.env
COPY --from=builder /app/builds-server /app/builds-server

# Ensure the binary is executable
RUN chmod +x /app/builds-server

# Pass environment variables
ARG APP_ENV
ENV APP_ENV=${APP_ENV}

# Run the binary
CMD ["/app/builds-server"]
