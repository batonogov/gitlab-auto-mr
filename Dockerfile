FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /gitlab-auto-mr

FROM alpine:3.21

COPY --from=builder /gitlab-auto-mr /usr/local/bin/gitlab-auto-mr

ENTRYPOINT ["gitlab-auto-mr"]