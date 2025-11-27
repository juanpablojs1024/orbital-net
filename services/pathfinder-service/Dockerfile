FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY pkg/ ./pkg/

COPY services/pathfinder-service/ ./services/pathfinder-service/
RUN CGO_ENABLED=0 GOOS=linux go build -o pathfinder-bin services/pathfinder-service/cmd/main.go

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/pathfinder-bin .

EXPOSE 50052
CMD ["./pathfinder-bin"]