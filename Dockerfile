FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /server /app/server
COPY --from=builder /app/proto /app/proto
EXPOSE 50051
CMD ["/app/server"]
