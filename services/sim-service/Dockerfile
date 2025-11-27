FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY pkg/ ./pkg/

COPY services/sim-service/ ./services/sim-service/
RUN CGO_ENABLED=0 GOOS=linux go build -o sim-bin services/sim-service/cmd/main.go

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/sim-bin .

EXPOSE 50051
CMD ["./sim-bin"]