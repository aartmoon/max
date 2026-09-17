FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gar-init ./cmd/gar-init

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app
USER app
COPY --from=build /out/gar-init /usr/local/bin/gar-init
ENTRYPOINT ["/usr/local/bin/gar-init"]
