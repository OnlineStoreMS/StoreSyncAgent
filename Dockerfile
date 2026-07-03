# syntax=docker/dockerfile:1

FROM node:20-alpine AS web-builder
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS api-builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /storesyncagent ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=api-builder /storesyncagent /app/storesyncagent
COPY --from=web-builder /src/web/dist /app/web/dist
COPY configs/config.example.yaml /app/configs/config.example.yaml
USER app
EXPOSE 8097
ENTRYPOINT ["/app/storesyncagent"]
CMD ["-config", "/app/configs/config.yaml", "-web-dist", "/app/web/dist"]
