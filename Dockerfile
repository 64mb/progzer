# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and progzer.go
COPY go.mod progzer.go ./

# Build for the current architecture
RUN go build -o progzer .

# Final stage - minimal image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/progzer /app/progzer

# Add documentation
COPY README.md /app/README.md

# Set the entrypoint to the binary
ENTRYPOINT ["/app/progzer"]