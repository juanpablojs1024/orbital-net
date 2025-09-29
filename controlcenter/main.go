package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"

	"satellite-coms/pkg/discovery/consul"
	discovery "satellite-coms/pkg/registry"
)

const serviceName = "controller"

var (
	ctx         = context.Background()
	redisClient *redis.Client
)

func main() {
	port := 8080

	log.Printf("🚀 Starting Controller service on port %d", port)

	// --- Connect to Redis ---
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Redis server
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	// --- Register service with Consul ---
	registry, err := consul.NewRegistry("localhost:8500") // Consul address
	if err != nil {
		log.Fatalf("❌ Failed to connect to Consul: %v", err)
	}

	hostname, _ := os.Hostname()
	instanceID := discovery.GenerateInstanceID(serviceName)
	serviceAddr := fmt.Sprintf("%s:%d", hostname, port)

	if err := registry.Register(ctx, instanceID, serviceName, serviceAddr); err != nil {
		log.Fatalf("❌ Failed to register in Consul: %v", err)
	}
	defer registry.Deregister(ctx, instanceID, serviceName)

	// Health reporting loop
	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("⚠️ Failed to report healthy state:", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// --- HTTP Handlers ---
	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/simulator-url", func(w http.ResponseWriter, r *http.Request) {
		url, err := getServiceURL("simulator", "simulator_url")
		if err != nil {
			http.Error(w, "Simulator service not found", http.StatusServiceUnavailable)
			log.Println("❌", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"url": "%s"}`, url)
	})

	http.HandleFunc("/positions", proxyToService("simulator", "simulator_url", "/positions"))
	http.HandleFunc("/visibility", proxyToService("simulator", "simulator_url", "/visibility"))
	http.HandleFunc("/send", proxyToService("communications", "communications_url", "/send"))

	log.Printf("🌐 Controller running at http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// --------------------
// Proxy helpers
// --------------------
func proxyToService(consulService, redisKey, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		baseURL, err := getServiceURL(consulService, redisKey)
		if err != nil {
			http.Error(w, "Failed to locate service", http.StatusServiceUnavailable)
			log.Println("❌", err)
			return
		}

		req, err := http.NewRequest(r.Method, baseURL+path, r.Body)
		if err != nil {
			http.Error(w, "Failed to build request", http.StatusInternalServerError)
			return
		}
		req.Header = r.Header.Clone()

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to reach service", http.StatusBadGateway)
			log.Println("❌", err)
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Println("❌ Failed to forward response:", err)
		}
	}
}

// --------------------
// Get service URL from Redis cache or Consul
// --------------------
func getServiceURL(consulService, redisKey string) (string, error) {
	// Try Redis first
	if url, err := redisClient.Get(ctx, redisKey).Result(); err == nil && url != "" {
		return url, nil
	}

	// Lookup in Consul
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return "", fmt.Errorf("failed to create Consul client: %w", err)
	}

	services, _, err := client.Health().Service(consulService, "", true, nil)
	if err != nil {
		return "", fmt.Errorf("failed to lookup %s in Consul: %w", consulService, err)
	}
	if len(services) == 0 {
		return "", fmt.Errorf("%s service not found", consulService)
	}

	svc := services[0].Service
	url := fmt.Sprintf("http://%s:%d", svc.Address, svc.Port)

	// Cache in Redis
	if err := redisClient.Set(ctx, redisKey, url, 30*time.Second).Err(); err != nil {
		log.Println("⚠️ Failed to cache service URL in Redis:", err)
	}

	return url, nil
}
