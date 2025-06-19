# Stage 1: The Build Stage
# We use a specific Go version for reproducibility.
# The 'alpine' tag provides a lightweight base image.
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files first. This allows Docker to cache the dependencies
# layer, speeding up subsequent builds if the dependencies haven't changed.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application.
# -o /fishdb2 specifies the output file name and path.
# We are creating a statically linked binary to ensure it runs in a minimal container.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /fishdb2 ./main.go

# ---

# Stage 2: The Final/Production Stage
# Start from a minimal, lightweight image. alpine is a great choice.
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the built binary from the 'builder' stage
COPY --from=builder /fishdb2 .

# Copy the assets the application needs at runtime.
# The Go binary will look for these directories relative to its execution path.
COPY --from=builder /app/db/ ./db/
COPY --from=builder /app/static/ ./static/
COPY --from=builder /app/templates/ ./templates/

# Expose the port the application will run on.
# AWS App Runner will use this port to route requests to the container.
# Make sure your Go application listens on this port (e.g., 8080).
EXPOSE 8080

# Command to run the application
CMD ["./fishdb2"]
