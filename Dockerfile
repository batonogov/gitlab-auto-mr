FROM golang:1.26.6-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o gitlab_auto_mr .

# Final stage
FROM alpine:3.24.1

RUN apk --no-cache --no-scripts add ca-certificates

# Run as a non-root user; the tool only needs to make outbound HTTPS calls
RUN adduser -D -H -u 10001 app

WORKDIR /app

# Copy the binary from builder stage (COPY --from preserves the executable bit)
COPY --from=builder /app/gitlab_auto_mr .

# Add to PATH
ENV PATH="/app:${PATH}"

USER app

CMD ["./gitlab_auto_mr"]
