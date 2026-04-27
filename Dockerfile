FROM node:22-alpine AS web-build
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web ./
RUN npm run build

FROM golang:1.25-alpine AS go-build
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /src/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/navbox ./cmd/navbox

FROM alpine:3.22
WORKDIR /app
RUN addgroup -S navbox && adduser -S navbox -G navbox && mkdir -p /app/data/uploads && chown -R navbox:navbox /app
COPY --from=go-build /out/navbox /app/navbox
USER navbox
ENV NAVBOX_HTTP_ADDR=:8037
ENV NAVBOX_UPLOAD_DIR=/app/data/uploads
EXPOSE 8037
ENTRYPOINT ["/app/navbox"]
