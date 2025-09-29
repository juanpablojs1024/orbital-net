package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"satellite-coms/pathfinder/model"
	"satellite-coms/pkg/discovery/consul"
	discovery "satellite-coms/pkg/registry"
)

const serviceName = "pathfinder"

var (
	ctx         = context.Background()
	redisClient *redis.Client
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8082, "Pathfinder service port")
	flag.Parse()

	log.Printf("🚀 Starting Pathfinder service on port %d", port)

	// 1️⃣ Connect to Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // no Docker container name
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	// 2️⃣ Register with Consul
	registry, err := consul.NewRegistry("localhost:8500") // no Docker container name
	if err != nil {
		log.Fatalf("❌ Failed to connect to Consul: %v", err)
	}

	instanceID := discovery.GenerateInstanceID(serviceName)
	serviceAddr := fmt.Sprintf("localhost:%d", port)

	if err := registry.Register(ctx, instanceID, serviceName, serviceAddr); err != nil {
		log.Fatalf("❌ Failed to register in Consul: %v", err)
	}
	defer registry.Deregister(ctx, instanceID, serviceName)

	// 3️⃣ Health check pinger
	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("⚠️ Failed to report healthy state:", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// 4️⃣ Subscribe to Redis events
	go subscribeToRedisEvents()

	// 5️⃣ HTTP routes
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		restrictedStr := r.URL.Query().Get("restricted")

		if start == "" || end == "" {
			http.Error(w, "start and end parameters are required", http.StatusBadRequest)
			return
		}

		restricted := make(map[string]bool)
		if restrictedStr != "" {
			for _, node := range strings.Split(restrictedStr, ",") {
				restricted[strings.TrimSpace(node)] = true
			}
		}

		g := model.CreateGraph() // Uses Redis + Consul internally
		path, found := g.WidestPath(start, end, restricted)

		w.Header().Set("Content-Type", "application/json")
		if found {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"path": path})
		} else {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("No path found from %s to %s", start, end),
			})
		}
	})

	// 6️⃣ Start HTTP server
	log.Printf("🌐 Pathfinder HTTP server listening on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
}

// Redis subscription
func subscribeToRedisEvents() {
	sub := redisClient.Subscribe(ctx, "simulation.step")
	defer sub.Close()
	log.Println("📡 Pathfinder subscribed to simulation.step")

	// Disabled to avoid spamming logs
	// for msg := range sub.Channel() {
	//     log.Println("🛰️ Step received:", msg.Payload)
	// }
}
