package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
)

// Redis client for caching service URLs
var redisClient *redis.Client

// in-memory cache (avoids hammering Consul)
var (
	serviceCache = make(map[string]string)
	cacheMutex   sync.RWMutex
)

// --- Redis initialization ---
func InitRedis(addr string) {
	redisClient = redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("⚠️ Redis ping failed: %v", err)
	}
}

// --- Simulator helpers ---

var simulatorBaseURL string
var simulatorOnce sync.Once

func getSimulatorBaseURL() (string, error) {
	var err error
	simulatorOnce.Do(func() {
		simulatorBaseURL, err = getServiceURLFromConsulInternal("simulator", "simulator_url")
	})
	if err != nil {
		return "", err
	}
	return simulatorBaseURL, nil
}

func fetchJSON(url string) (interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed interface{}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

// FetchVisibility returns the visibility matrix from the simulator
func FetchVisibility() []interface{} {
	baseURL, err := getSimulatorBaseURL()
	if err != nil {
		log.Fatalf("❌ Failed to discover simulator service: %v", err)
	}

	data, err := fetchJSON(fmt.Sprintf("%s/visibility", baseURL))
	if err != nil {
		log.Fatalf("❌ Failed to fetch visibility: %v", err)
	}

	matrix, ok := data.([]interface{})
	if !ok {
		log.Fatal("❌ Failed to cast visibility to []interface{}")
	}
	return matrix
}

// FetchNodes returns all simulator nodes
func FetchNodes() []interface{} {
	baseURL, err := getSimulatorBaseURL()
	if err != nil {
		log.Fatalf("❌ Failed to discover simulator service: %v", err)
	}

	data, err := fetchJSON(fmt.Sprintf("%s/positions", baseURL))
	if err != nil {
		log.Fatalf("❌ Failed to fetch positions: %v", err)
	}

	items, ok := data.([]interface{})
	if !ok {
		log.Fatal("❌ Expected JSON array at top level")
	}
	return items
}

// --- Generic service discovery ---

func GetServiceURLFromConsul(consulService, redisKey string, ctx context.Context) (string, error) {
	// Check in-memory cache
	cacheMutex.RLock()
	if url, ok := serviceCache[consulService]; ok && url != "" {
		cacheMutex.RUnlock()
		return url, nil
	}
	cacheMutex.RUnlock()

	// Check Redis
	if redisClient != nil {
		if url, err := redisClient.Get(ctx, redisKey).Result(); err == nil && url != "" {
			cacheMutex.Lock()
			serviceCache[consulService] = url
			cacheMutex.Unlock()
			return url, nil
		}
	}

	// Lookup in Consul
	url, err := getServiceURLFromConsulInternal(consulService, redisKey)
	if err != nil {
		return "", err
	}

	return url, nil
}

// internal helper to query Consul and cache results
func getServiceURLFromConsulInternal(consulService, redisKey string) (string, error) {
	config := api.DefaultConfig()
	config.Address = "http://localhost:8500"
	client, err := api.NewClient(config)
	if err != nil {
		return "", fmt.Errorf("failed to create Consul client: %w", err)
	}

	var svcAddr string
	for i := 0; i < 5; i++ { // retry a few times
		services, _, err := client.Health().Service(consulService, "", true, nil)
		if err != nil {
			log.Printf("⚠️ Consul lookup failed: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if len(services) > 0 {
			svc := services[0].Service
			svcAddr = fmt.Sprintf("http://%s:%d", svc.Address, svc.Port)
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if svcAddr == "" {
		return "", fmt.Errorf("%s service not found in Consul", consulService)
	}

	// Cache in memory
	cacheMutex.Lock()
	serviceCache[consulService] = svcAddr
	cacheMutex.Unlock()

	// Cache in Redis
	if redisClient != nil {
		if err := redisClient.Set(context.Background(), redisKey, svcAddr, 5*time.Minute).Err(); err != nil {
			log.Println("⚠️ Failed to cache service URL in Redis:", err)
		}
	}

	return svcAddr, nil
}
