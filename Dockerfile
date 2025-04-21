FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /gitlab_auto_mr

FROM alpine:3.21

COPY --from=builder /gitlab_auto_mr /usr/local/bin/gitlab_auto_mr

ENTRYPOINT ["gitlab_auto_mr"]
