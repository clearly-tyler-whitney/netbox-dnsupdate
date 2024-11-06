FROM golang:1.23-alpine AS builder

# Install git for fetching dependencies
RUN apk add --no-cache git

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o netbox-dnsupdate *.go

# -----------------------------
# Stage 2: Create a minimal image
# -----------------------------
FROM alpine:latest

# Install CA certificates (if needed)
RUN apk add --no-cache ca-certificates bind-tools

# Set the working directory
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/netbox-dnsupdate .

# Expose the port your application listens on
EXPOSE 8080

#Set environment variables (optional)
#ENV LISTEN_ADDRESS=":8080"
#ENV BIND_SERVER_ADDRESS="127.0.0.1:53"
#ENV TSIG_KEY_FILE="/etc/nsupdate.key"
#ENV LOG_LEVEL="info"
#ENV LOG_FORMAT="logfmt"

# Set the entrypoint
ENTRYPOINT ["./netbox-dnsupdate"]
