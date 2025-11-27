package config

import "os"

var ServerHost = getEnv("SERVER_HOST", "localhost")

const (
	GRPC_PORT         = "50051"
	SERVER_TICK_DELAY = 10
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetAddress() string {
	return ServerHost + ":" + GRPC_PORT
}
