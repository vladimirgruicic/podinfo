# Build stage: Compile Go binary
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o podinfo ./main.go

# Runtime stage: Minimal image, non-root user
FROM alpine:3.18
RUN apk --no-cache add ca-certificates wget && \
    addgroup -g 1001 -S podgroup && \
    adduser -S poduser -u 1001 -G podgroup
WORKDIR /app
COPY --from=builder /app/podinfo /app/podinfo
RUN chown poduser:podgroup /app/podinfo
USER poduser
EXPOSE 9898
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9898/healthz || exit 1
ENTRYPOINT ["/app/podinfo"]