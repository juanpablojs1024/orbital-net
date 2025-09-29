package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"

	"satellite-coms/pkg/discovery/consul"
	discovery "satellite-coms/pkg/registry"
)

const serviceName = "message-db"

var (
	db          *sql.DB
	ctx         = context.Background()
	redisClient *redis.Client
)

type Message struct {
	ID         int    `json:"id"`
	RecieverID string `json:"reciever_id"`
	SenderID   string `json:"sender_id"`
	Payload    string `json:"payload"`
}

func main() {
	var port int
	flag.IntVar(&port, "port", 8084, "Message DB service port")
	flag.Parse()

	log.Printf("🚀 Starting Message DB service on port %d", port)

	// Connect to Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	// Open SQLite DB
	var err error
	db, err = sql.Open("sqlite3", "./database/messages.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize schema
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		reciever_id TEXT NOT NULL,
		sender_id TEXT NOT NULL,
		payload TEXT NOT NULL
	);`)
	if err != nil {
		log.Fatal(err)
	}

	// Register service in Consul
	registry, err := consul.NewRegistry("localhost:8500")
	if err != nil {
		log.Fatalf("❌ Failed to connect to Consul: %v", err)
	}
	instanceID := discovery.GenerateInstanceID(serviceName)
	serviceAddr := fmt.Sprintf("%s:%d", serviceName, port)

	if err := registry.Register(ctx, instanceID, serviceName, serviceAddr); err != nil {
		log.Fatalf("❌ Failed to register service in Consul: %v", err)
	}
	defer registry.Deregister(ctx, instanceID, serviceName)

	// Start health reporting to Consul
	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("⚠️ Failed to report healthy state:", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// HTTP Handlers
	http.HandleFunc("/messages", handleMessages)
	http.HandleFunc("/messages/", handleMessageByID)

	log.Printf("📦 Message DB running at http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, reciever_id, sender_id, payload FROM messages")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		var msgs []Message
		for rows.Next() {
			var m Message
			if err := rows.Scan(&m.ID, &m.RecieverID, &m.SenderID, &m.Payload); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			msgs = append(msgs, m)
		}
		json.NewEncoder(w).Encode(msgs)

	case http.MethodPost:
		var m Message
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}
		res, err := db.Exec("INSERT INTO messages (reciever_id, sender_id, payload) VALUES (?, ?, ?)",
			m.RecieverID, m.SenderID, m.Payload)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		id, _ := res.LastInsertId()
		m.ID = int(id)
		json.NewEncoder(w).Encode(m)

	default:
		http.Error(w, "Method not allowed", 405)
	}
}

func handleMessageByID(w http.ResponseWriter, r *http.Request) {
	var id int
	_, err := fmt.Sscanf(r.URL.Path, "/messages/%d", &id)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var m Message
		err := db.QueryRow("SELECT id, reciever_id, sender_id, payload FROM messages WHERE id = ?", id).
			Scan(&m.ID, &m.RecieverID, &m.SenderID, &m.Payload)
		if err != nil {
			http.Error(w, "Not found", 404)
			return
		}
		json.NewEncoder(w).Encode(m)

	case http.MethodDelete:
		_, err := db.Exec("DELETE FROM messages WHERE id = ?", id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(204)

	default:
		http.Error(w, "Method not allowed", 405)
	}
}
