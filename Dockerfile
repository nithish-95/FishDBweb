# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app. CGO_ENABLED=0 is important for building a static binary
# that can run in a minimal container from scratch.
# -o /app/fishdb builds the application into an executable named fishdb
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/fishdb .

# Stage 2: Create the final, minimal image
FROM alpine:latest

WORKDIR /app

# Copy the static assets and templates from the builder stage
COPY --from=builder /app/db/ ./db/
COPY --from=builder /app/templates/ ./templates/
COPY --from=builder /app/static/ ./static/

# Copy the built binary from the builder stage
COPY --from=builder /app/fishdb .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./fishdb"]