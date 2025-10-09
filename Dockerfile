FROM golang:1.25.2-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gitlab_auto_mr .

# Final stage
FROM alpine:3.22.2

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/gitlab_auto_mr .

# Make it executable
RUN chmod +x ./gitlab_auto_mr

# Add to PATH
ENV PATH="/app:${PATH}"

CMD ["./gitlab_auto_mr"]
