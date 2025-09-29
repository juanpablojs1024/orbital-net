package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"satellite-coms/communications/model"
	"satellite-coms/pkg/discovery/consul"
	discovery "satellite-coms/pkg/registry"

	"github.com/redis/go-redis/v9"
)

const serviceName = "communications"

var (
	ctx          = context.Background()
	redisClient  *redis.Client
	logicalNodes []*model.LogicalNode
	restrictions = make(map[string]struct{})
)

func main() {
	// Allow port to be specified
	var port int
	flag.IntVar(&port, "port", 8083, "Communications service port")
	flag.Parse()

	log.Printf("🚀 Starting Communications service on port %d", port)

	// 1. Connect to Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	// 2. Register with Consul
	registry, err := consul.NewRegistry("localhost:8500")
	if err != nil {
		log.Fatalf("❌ Failed to connect to Consul: %v", err)
	}
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("localhost:%d", port)); err != nil {
		log.Fatalf("❌ Failed to register in Consul: %v", err)
	}
	defer registry.Deregister(ctx, instanceID, serviceName)

	// 3. Health check pinger
	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("⚠️ Failed to report healthy state:", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// 4. Prepare initial state
	logicalNodes = model.GetLogicalNodes()

	// 5. Start HTTP server (health + send endpoint)
	go startHTTPServer(port)

	// 6. Start Redis-based simulation logic
	runRedisLoop()
}

func startHTTPServer(port int) {
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/send", handleSendMessage)

	log.Printf("🌐 HTTP server listening on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	type SendRequest struct {
		Origin      string `json:"origin"`
		Destination string `json:"destination"`
		Message     string `json:"message"`
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Origin == "" || req.Destination == "" || req.Message == "" {
		http.Error(w, "Missing origin, destination, or message", http.StatusBadRequest)
		return
	}

	model.SendMessage(req.Origin, req.Destination, req.Message, logicalNodes, restrictions)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message sent"))
}

func runRedisLoop() {
	// Subscribe to simulation.step
	sub := redisClient.Subscribe(ctx, "simulation.step")
	defer sub.Close()
	log.Println("📡 Subscribed to simulation.step")

	for msg := range sub.Channel() {
		fmt.Print("\033[2J\033[H")
		log.Println("🛰️ Received:", msg.Payload)

		model.CommunicationProtocol(logicalNodes, restrictions)

		// Example hardcoded message (can be removed if using /send externally)
		//model.SendMessage("srv_6c3a7", "srv_70f8b", "Hello", logicalNodes, restrictions)
	}
}
