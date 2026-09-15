FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/magic-stream-server .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app
WORKDIR /app
COPY --from=builder /out/magic-stream-server /app/magic-stream-server

USER app
EXPOSE 8080
ENTRYPOINT ["/app/magic-stream-server"]
