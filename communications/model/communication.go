package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
)

// --- Globals ---
var (
	ctx         = context.Background()
	redisClient *redis.Client // initialize this in main or init function
)

// --- Consul service discovery (localhost only) ---
func GetServiceURL(serviceName string) (string, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return "", fmt.Errorf("failed to create Consul client: %w", err)
	}

	for i := 0; i < 5; i++ {
		services, _, err := client.Health().Service(serviceName, "", true, nil)
		if err != nil {
			log.Printf("⚠️ Consul lookup failed (%d/5): %v", i+1, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if len(services) > 0 {
			svc := services[0].Service
			host := svc.Address
			if host == "" || host == "message-db" {
				host = "localhost" // force localhost
			}
			url := fmt.Sprintf("http://%s:%d", host, svc.Port)
			return url, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", fmt.Errorf("%s service not found in Consul", serviceName)
}

// --- Communication protocol ---
func CommunicationProtocol(logicalNodes []*LogicalNode, restrictions map[string]struct{}) {
	log.Println("🧠 Logical nodes:")
	for _, ln := range logicalNodes {
		fmt.Printf(" - %s [%s]{%s} |", ln.ID, ln.Message.Payload, ln.State)
		for _, ins := range ln.Memory {
			fmt.Printf(" %s", ins)
		}
		fmt.Println(" |")

		if ln.State != "wsc" {
			continue
		}

		path := ln.GetPath(restrictions)
		if len(path) < 2 {
			continue
		}

		tempLn := GetLogicalNodeById(path[1], logicalNodes)
		if tempLn == nil {
			continue
		}

		restrictions[tempLn.ID] = struct{}{}
		tempLn.Message = ln.Message

		if ln.Message.Objective != nil && tempLn.ID == ln.Message.Objective.ID {
			sendMessageToDB(ln, tempLn)
			tempLn.State = ""
			tempLn.Message = Instruction{}
			delete(restrictions, tempLn.ID)
		} else {
			tempLn.State = "wsc"
		}

		delete(restrictions, ln.ID)
		ln.Message = Instruction{}
		ln.State = ""
	}
}

// --- Send message to message-db service ---
func sendMessageToDB(origin, target *LogicalNode) {
	stripID := func(fullID string) string {
		if idx := strings.Index(fullID, ":"); idx != -1 {
			return fullID[:idx]
		}
		return fullID
	}

	msg := map[string]string{
		"reciever_id": stripID(target.ID),
		"sender_id":   stripID(origin.ID),
		"payload":     target.Message.Payload,
	}

	body, _ := json.Marshal(msg)

	url, err := GetServiceURL("message-db")
	if err != nil {
		log.Printf("❌ Could not discover DB service: %v", err)
		return
	}

	for i := 0; i < 3; i++ {
		resp, err := http.Post(url+"/messages", "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("⚠️ Failed to send message (attempt %d): %v", i+1, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		resp.Body.Close()
		log.Printf("✅ Message sent to DB: %s -> %s", stripID(origin.ID), stripID(target.ID))
		break
	}
}

// --- SendMessage utility ---
func SendMessage(originID, destinationID, message string, logicalNodes []*LogicalNode, restrictions map[string]struct{}) {
	origin := getFreePortNode(originID, logicalNodes, restrictions)
	destination := getFreePortNode(destinationID, logicalNodes, restrictions)

	if origin == nil || destination == nil {
		log.Printf("❌ Could not find free ports for origin (%s) or destination (%s)", originID, destinationID)
		return
	}

	origin.SendInstruction(destination, message)
}

// --- Helper: find free port logical node ---
func getFreePortNode(baseID string, logicalNodes []*LogicalNode, restrictions map[string]struct{}) *LogicalNode {
	for _, ln := range logicalNodes {
		if matchesBaseID(ln.ID, baseID) {
			if _, restricted := restrictions[ln.ID]; !restricted {
				return ln
			}
		}
	}
	return nil
}

func matchesBaseID(fullID, baseID string) bool {
	return len(fullID) >= len(baseID) && fullID[:len(baseID)] == baseID
}
