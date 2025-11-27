FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY pkg/ ./pkg/

COPY services/storage-service/ ./services/storage-service/
RUN CGO_ENABLED=0 GOOS=linux go build -o storage-bin services/storage-service/cmd/main.go

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/storage-bin .

EXPOSE 50054
CMD ["./storage-bin"]