FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -o /server ./cmd/server
FROM alpine:3.23
RUN adduser -D app
USER app
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
