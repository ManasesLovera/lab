# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o monitor .

# Final stage
FROM alpine:3.20
# coreutils/findutils: GNU du (-b) and find. docker-cli: `docker system df`.
RUN apk --no-cache add ca-certificates coreutils findutils docker-cli
WORKDIR /app
COPY --from=builder /app/monitor .
EXPOSE 8080
CMD ["./monitor"]
