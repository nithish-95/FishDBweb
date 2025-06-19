# Stage 1: Build the Go application
FROM golang:1.22.5-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
# This step is cached if dependencies haven't changed
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 is important for creating a statically linked executable
# which is ideal for a minimal base image like 'scratch' or 'alpine'.
# -o main specifies the output executable name as 'main'
# ./ means build the current directory
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o main ./main.go

# Stage 2: Create the final, lean image
# Using alpine as it's small but includes basic utilities if needed
FROM alpine:latest

WORKDIR /app

# Copy the compiled application from the builder stage
COPY --from=builder /app/main .

# Copy the database (JSON) files
COPY db ./db

# Copy the HTML templates
COPY templates ./templates

# Copy the static assets (images, CSS, JS, etc.)
# This includes the nested 'images' and 'map' directories correctly.
COPY static ./static

# Expose the port your Go application listens on (8080 by default)
EXPOSE 8080

# Command to run the application when the container starts
CMD ["./main"]
