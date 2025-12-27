ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
ARG BUILD_VERSION=unknown
RUN go build -v -ldflags="-X 'main.appVersion=${BUILD_VERSION}'" -o /run-app ./cmd


FROM debian:bookworm

COPY --from=builder /run-app /usr/local/bin/

# Listen on all interfaces in production, not just localhost
ENV ME_WEB_ADDR=0.0.0.0:8080
# Enable static file caching in production
ENV ME_WEB_DISABLE_STATIC_CACHE=false

CMD ["run-app"]
