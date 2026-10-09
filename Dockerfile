# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/healthcheck ./cmd/healthcheck

FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /out/server /app/server
COPY --from=builder /out/healthcheck /app/healthcheck
COPY conf/ /app/conf/
COPY config.yaml /app/config.yaml
USER app
EXPOSE 8999
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 CMD ["/app/healthcheck"]
ENTRYPOINT ["/app/server"]
