FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY pkg/ ./pkg/

COPY services/communications-service/ ./services/communications-service/
RUN CGO_ENABLED=0 GOOS=linux go build -o communications-bin services/communications-service/cmd/main.go

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/communications-bin .

EXPOSE 50053
CMD ["./communications-bin"]