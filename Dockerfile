ARG GO_VERSION=1.24
ARG ALPINE_VERSION=3.22
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

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
FROM alpine:${ALPINE_VERSION}

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/gitlab_auto_mr .

# Make it executable
RUN chmod +x ./gitlab_auto_mr

# Add to PATH
ENV PATH="/app:${PATH}"

CMD ["./gitlab_auto_mr"]
