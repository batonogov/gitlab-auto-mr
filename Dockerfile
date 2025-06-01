FROM golang:1.21-alpine AS builder

# Build arguments for version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Install git for go mod download
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy source code (no external dependencies to cache)
COPY . .

# Build the application with version information
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags \"-static\" -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -a -installsuffix cgo \
    -o gitlab_auto_mr .

FROM alpine:3.22

LABEL maintainer="gitlab-auto-mr" \
      description="GitLab Auto MR - automatically creates merge requests in GitLab"

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /home/appuser

# Copy the binary from builder stage
COPY --from=builder /app/gitlab_auto_mr /usr/local/bin/gitlab_auto_mr

# Make sure the binary is executable
RUN chmod +x /usr/local/bin/gitlab_auto_mr

# Switch to non-root user
USER appuser

ENTRYPOINT ["gitlab_auto_mr"]
