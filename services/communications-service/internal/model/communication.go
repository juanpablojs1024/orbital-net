package model

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	pbPath "orbital-net/pkg/proto/pathfinder"
	pbStorage "orbital-net/pkg/proto/storage"
)

func CommunicationProtocol(logicalNodes []*LogicalNode, restrictions map[string]struct{}, pfClient pbPath.PathfinderServiceClient, storageClient pbStorage.StorageServiceClient) {
	for _, ln := range logicalNodes {
		if ln.State != "wsc" {
			continue
		}

		path := ln.GetPath(pfClient, restrictions)
		if len(path) < 2 {
			continue
		}

		nextNodePhysicalID := path[1]
		tempLn := getFreePortNode(nextNodePhysicalID, logicalNodes)

		if tempLn == nil {
			continue
		}

		if _, booked := restrictions[tempLn.ID]; booked {
			continue
		}

		restrictions[tempLn.ID] = struct{}{}
		tempLn.Message = ln.Message

		if ln.Message.Objective != nil && strings.HasPrefix(tempLn.ID, ln.Message.Objective.ID) {
			log.Printf("✅ ENTREGADO: %s -> %s", ln.Message.Origin.Name, tempLn.Name)

			if storageClient != nil {
				go func(origin, dest, payload string) {
					_, err := storageClient.SaveMessage(context.Background(), &pbStorage.SaveRequest{
						SenderId:   origin,
						ReceiverId: dest,
						Payload:    payload,
						Timestamp:  time.Now().Unix(),
					})
					if err != nil {
						log.Printf("⚠️ Error guardando en DB: %v", err)
					}
				}(ln.Message.Origin.Name, tempLn.Name, ln.Message.Payload)
			}

			tempLn.State = ""
			tempLn.Message = Instruction{}
			delete(restrictions, tempLn.ID)
		} else {
			tempLn.State = "wsc"
			fmt.Printf("🛰️ Salto: %s -> %s\n", ln.Name, tempLn.Name)
		}

		delete(restrictions, ln.ID)
		ln.Message = Instruction{}
		ln.State = ""
	}
}

func SendMessage(originBaseID, destinationBaseID, message string, logicalNodes []*LogicalNode) error {
	if !nodeExists(originBaseID, logicalNodes) {
		return fmt.Errorf("origen '%s' no existe", originBaseID)
	}
	if !nodeExists(destinationBaseID, logicalNodes) {
		return fmt.Errorf("destino '%s' no existe", destinationBaseID)
	}

	origin := getFreePortNode(originBaseID, logicalNodes)
	if origin == nil {
		return fmt.Errorf("el nodo origen '%s' está saturado (todos puertos ocupados)", originBaseID)
	}

	destination := getAnyPortNode(destinationBaseID, logicalNodes)

	log.Printf("📨 Nuevo mensaje encolado: %s -> %s", origin.Name, destination.Name)
	origin.SendInstruction(destination, message)
	return nil
}

func nodeExists(baseID string, logicalNodes []*LogicalNode) bool {
	prefix := strings.Split(baseID, ":")[0] + ":"
	for _, ln := range logicalNodes {
		if strings.HasPrefix(ln.ID, prefix) {
			return true
		}
	}
	return false
}

func getFreePortNode(baseID string, logicalNodes []*LogicalNode) *LogicalNode {
	cleanBaseID := strings.Split(baseID, ":")[0]
	prefix := cleanBaseID + ":"
	for _, ln := range logicalNodes {
		if strings.HasPrefix(ln.ID, prefix) && ln.State == "" {
			return ln
		}
	}
	return nil
}

func getAnyPortNode(baseID string, logicalNodes []*LogicalNode) *LogicalNode {
	cleanBaseID := strings.Split(baseID, ":")[0]
	prefix := cleanBaseID + ":"
	for _, ln := range logicalNodes {
		if strings.HasPrefix(ln.ID, prefix) {
			return ln
		}
	}
	return nil
}

func GetLogicalNodeById(id string, logicalNodes []*LogicalNode) *LogicalNode {
	for _, ln := range logicalNodes {
		if ln.ID == id {
			return ln
		}
	}
	return nil
}
