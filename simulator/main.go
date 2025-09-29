package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"satellite-coms/pkg/discovery/consul"
	discovery "satellite-coms/pkg/registry"
	"satellite-coms/simulator/handler"
	"satellite-coms/simulator/simulation"
)

const serviceName = "simulator"

var (
	ctx         = context.Background()
	redisClient *redis.Client
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8081, "API handler port")
	flag.Parse()

	log.Printf("🚀 Starting simulator service on port %d", port)

	// 1️⃣ Connect to Redis (retry until ready)
	for {
		redisClient = redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Println("⚠️ Redis not ready, retrying in 2s...")
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	// 2️⃣ Initialize simulation state
	simulation.InitSimulation()

	// 3️⃣ Register with Consul using container hostname
	registry, err := consul.NewRegistry("localhost:8500")
	if err != nil {
		log.Fatalf("❌ Failed to connect to Consul: %v", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("❌ Failed to get container hostname: %v", err)
	}

	instanceID := discovery.GenerateInstanceID(serviceName)
	serviceAddr := fmt.Sprintf("%s:%d", hostname, port)
	if err := registry.Register(ctx, instanceID, serviceName, serviceAddr); err != nil {
		log.Fatalf("❌ Failed to register in Consul: %v", err)
	}
	defer registry.Deregister(ctx, instanceID, serviceName)

	// 4️⃣ Start health reporting loop
	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("⚠️ Failed to report healthy state:", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// 5️⃣ Start automatic simulation steps
	go func() {
		for {
			stepAndPublishHandler(dummyResponseWriter{}, nil)
			time.Sleep(5 * time.Millisecond) // Adjust step speed
		}
	}()

	// 6️⃣ HTTP Handlers
	http.HandleFunc("/positions", handler.GetPositionsHandler)
	http.HandleFunc("/visibility", handler.GetVisibilityMatrixHandler)
	http.HandleFunc("/step", stepAndPublishHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	log.Printf("🌐 Simulator HTTP server listening on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// stepAndPublishHandler executes a simulation step and publishes an event to Redis
func stepAndPublishHandler(w http.ResponseWriter, r *http.Request) {
	handler.StepHandler(w, r)

	err := redisClient.Publish(ctx, "simulation.step", "step executed").Err()
	if err != nil {
		log.Printf("❌ Failed to publish simulation.step event: %v", err)
	}
}

// dummyResponseWriter allows calling the handler without an actual HTTP request
type dummyResponseWriter struct{}

func (d dummyResponseWriter) Header() http.Header        { return http.Header{} }
func (d dummyResponseWriter) Write([]byte) (int, error)  { return 0, nil }
func (d dummyResponseWriter) WriteHeader(statusCode int) {}
