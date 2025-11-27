package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"orbital-net/pkg/config"
	pbComs "orbital-net/pkg/proto/communications"
	pbSim "orbital-net/pkg/proto/sim"
	pbStorage "orbital-net/pkg/proto/storage"
)

//go:embed web/index.html
var indexHTML []byte

var (
	stateMu     sync.RWMutex
	latestNodes []interface{}
)

func main() {
	simConn := connectGRPC(config.GetAddress())
	comsConn := connectGRPC(getServiceAddr("communications-server", "50053"))
	storageConn := connectGRPC(getServiceAddr("storage-server", "50054"))

	simClient := pbSim.NewSimulationServiceClient(simConn)
	comsClient := pbComs.NewCommunicationsServiceClient(comsConn)
	storageClient := pbStorage.NewStorageServiceClient(storageConn)

	go consumeSimulation(simClient)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexHTML)
	})

	http.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		stateMu.RLock()
		defer stateMu.RUnlock()
		json.NewEncoder(w).Encode(latestNodes)
	})

	http.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		resp, err := storageClient.GetLogs(context.Background(), &pbStorage.Empty{})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(resp.Entries)
	})

	http.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Origin      string `json:"origin"`
			Destination string `json:"destination"`
			Message     string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}

		resp, err := comsClient.SendMessage(context.Background(), &pbComs.MessageRequest{
			OriginId:      req.Origin,
			DestinationId: req.Destination,
			Payload:       req.Message,
		})

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/api/create-node", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string  `json:"name"`
			Altitude float64 `json:"altitude"`
			Angle    float64 `json:"angle"`
			Type     string  `json:"type"`
			Ports    int     `json:"ports"`
			Gen      int     `json:"gen"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}

		nodeType := pbSim.NodeType_SATELLITE
		if req.Type == "station" {
			nodeType = pbSim.NodeType_STATION
		}

		resp, err := simClient.CreateNode(context.Background(), &pbSim.CreateNodeRequest{
			Name:           req.Name,
			Altitude:       req.Altitude,
			Angle:          req.Angle,
			Type:           nodeType,
			PortQuantity:   int32(req.Ports),
			PortGeneration: int32(req.Gen),
		})

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/api/delete-node", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Id string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}

		resp, err := simClient.DeleteNode(context.Background(), &pbSim.DeleteNodeRequest{NodeId: req.Id})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/api/clear-all", func(w http.ResponseWriter, r *http.Request) {
		_, err := simClient.ClearSimulation(context.Background(), &pbSim.Empty{})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte(`{"success": true}`))
	})

	log.Println("🎮 Control Center en puerto 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func consumeSimulation(client pbSim.SimulationServiceClient) {
	for {
		stream, err := client.StreamSimulation(context.Background(), &pbSim.Empty{})
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		for {
			data, err := stream.Recv()
			if err != nil {
				break
			}
			var nodes []interface{}
			for _, n := range data.Nodes {
				nodes = append(nodes, map[string]interface{}{
					"id":    n.Id,
					"name":  n.Name,
					"x":     n.X,
					"y":     n.Y,
					"peers": n.VisiblePeers,
				})
			}
			stateMu.Lock()
			latestNodes = nodes
			stateMu.Unlock()
		}
	}
}

func connectGRPC(addr string) *grpc.ClientConn {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("⚠️ Error conectando a %s: %v", addr, err)
	}
	return conn
}

func getServiceAddr(name, port string) string {
	if config.GetAddress() == "localhost:50051" {
		return "localhost:" + port
	}
	return name + ":" + port
}
