# Build stage
FROM golang:1.21-alpine AS build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

# Runtime stage
FROM alpine:3.19
RUN adduser -D -H app && apk add --no-cache ca-certificates
COPY --from=build /out/server /usr/local/bin/server
EXPOSE 8080
USER app
ENTRYPOINT ["/usr/local/bin/server"]

