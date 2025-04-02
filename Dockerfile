# First Stage: Build the Go application
FROM golang:1.24.1-alpine3.20 AS builder

WORKDIR /app

# Copy only go.mod (skip go.sum if missing)
COPY go.mod ./
RUN [ -f go.sum ] && COPY go.sum ./ || true

# Download dependencies (ignores missing go.sum)
RUN go mod tidy

# Copy the rest of the source code
COPY . .

# Compile the application
RUN go build -o main main/main.go

# Second Stage: Create a minimal final image
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 8080
CMD ["./main"]
