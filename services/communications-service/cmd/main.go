package main

import (
	"context"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"orbital-net/pkg/config"

	pbComs "orbital-net/pkg/proto/communications"
	pbPath "orbital-net/pkg/proto/pathfinder"
	pbSim "orbital-net/pkg/proto/sim"
	pbStorage "orbital-net/pkg/proto/storage"

	"orbital-net/services/communications-service/internal/model"
)

const COMS_PORT = "50053"

type PendingMessage struct {
	OriginID      string
	DestinationID string
	Payload       string
}

type ComsServer struct {
	pbComs.UnimplementedCommunicationsServiceServer

	mu           sync.Mutex
	logicalNodes []*model.LogicalNode
	restrictions map[string]struct{}
	messageQueue []PendingMessage
}

func (s *ComsServer) SendMessage(ctx context.Context, req *pbComs.MessageRequest) (*pbComs.MessageResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.logicalNodes) == 0 {
		return &pbComs.MessageResponse{Success: false, Message: "Sistema no inicializado"}, nil
	}

	err := model.SendMessage(req.OriginId, req.DestinationId, req.Payload, s.logicalNodes)

	if err != nil {
		if strings.Contains(err.Error(), "saturado") || strings.Contains(err.Error(), "ocupados") {
			s.messageQueue = append(s.messageQueue, PendingMessage{
				OriginID:      req.OriginId,
				DestinationID: req.DestinationId,
				Payload:       req.Payload,
			})
			log.Printf("📥 Buffer: Mensaje encolado (%d en espera).", len(s.messageQueue))
			return &pbComs.MessageResponse{Success: true, Message: "Encolado (Red saturada)"}, nil
		}
		return &pbComs.MessageResponse{Success: false, Message: err.Error()}, nil
	}

	return &pbComs.MessageResponse{Success: true, Message: "Enviado"}, nil
}

func (s *ComsServer) processQueue() {
	if len(s.messageQueue) == 0 {
		return
	}

	var remainingMessages []PendingMessage
	processedCount := 0

	for _, msg := range s.messageQueue {
		err := model.SendMessage(msg.OriginID, msg.DestinationID, msg.Payload, s.logicalNodes)
		if err == nil {
			processedCount++
		} else {
			remainingMessages = append(remainingMessages, msg)
		}
	}

	if processedCount > 0 {
		log.Printf("📤 Buffer: %d mensajes procesados, %d quedan en espera.", processedCount, len(remainingMessages))
	}

	s.messageQueue = remainingMessages
}

func main() {
	lis, err := net.Listen("tcp", ":"+COMS_PORT)
	if err != nil {
		log.Fatalf("Fallo puerto: %v", err)
	}

	s := &ComsServer{
		restrictions: make(map[string]struct{}),
		messageQueue: make([]PendingMessage, 0),
	}

	grpcServer := grpc.NewServer()
	pbComs.RegisterCommunicationsServiceServer(grpcServer, s)

	go s.runSimulationLoop()

	log.Printf("📡 Communications Service corriendo en puerto %s", COMS_PORT)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Fallo serve: %v", err)
	}
}

func (s *ComsServer) runSimulationLoop() {
	targetSim := config.GetAddress()
	targetPath := "pathfinder-server:50052"
	targetStorage := "storage-server:50054"

	if targetSim == "localhost:50051" {
		targetPath = "localhost:50052"
		targetStorage = "localhost:50054"
	}

	var pfClient pbPath.PathfinderServiceClient
	for {
		connPath, err := grpc.NewClient(targetPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			pfClient = pbPath.NewPathfinderServiceClient(connPath)
			break
		}
		log.Println("Esperando a Pathfinder...")
		time.Sleep(2 * time.Second)
	}

	var storageClient pbStorage.StorageServiceClient
	connStorage, err := grpc.NewClient(targetStorage, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		storageClient = pbStorage.NewStorageServiceClient(connStorage)
	}

	for {
		connSim, err := grpc.NewClient(targetSim, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		simClient := pbSim.NewSimulationServiceClient(connSim)
		stream, err := simClient.StreamSimulation(context.Background(), &pbSim.Empty{})
		if err != nil {
			connSim.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		log.Println("✅ Conectado al flujo de simulación.")

		for {
			state, err := stream.Recv()
			if err == io.EOF || err != nil {
				log.Println("Desconectado del simulador.")
				break
			}

			s.mu.Lock()

			s.logicalNodes = model.SyncLogicalNodes(s.logicalNodes, state.Nodes)

			s.processQueue()
			model.CommunicationProtocol(s.logicalNodes, s.restrictions, pfClient, storageClient)

			s.mu.Unlock()
		}
		connSim.Close()
		time.Sleep(1 * time.Second)
	}
}
