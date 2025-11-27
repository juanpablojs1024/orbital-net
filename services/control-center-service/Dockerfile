FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY pkg/ ./pkg/
COPY services/control-center-service/ ./services/control-center-service/

RUN CGO_ENABLED=0 GOOS=linux go build -o control-center-bin services/control-center-service/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/control-center-bin .
EXPOSE 8080
CMD ["./control-center-bin"]