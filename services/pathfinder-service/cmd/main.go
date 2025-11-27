package main

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"orbital-net/pkg/config"

	pbPath "orbital-net/pkg/proto/pathfinder"
	pbSim "orbital-net/pkg/proto/sim"

	"orbital-net/services/pathfinder-service/internal/model"
)

const PATHFINDER_PORT = "50052"

type PathfinderServer struct {
	pbPath.UnimplementedPathfinderServiceServer

	mu          sync.RWMutex
	latestGraph *model.Graph
}

func (s *PathfinderServer) FindPath(ctx context.Context, req *pbPath.PathRequest) (*pbPath.PathResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latestGraph == nil {
		return &pbPath.PathResponse{Found: false, ErrorMessage: "Graph not initialized yet"}, nil
	}

	log.Printf("Calculando ruta de %s a %s", req.StartNodeId, req.EndNodeId)
	path, found := s.latestGraph.FindShortestPath(req.StartNodeId, req.EndNodeId)

	if !found {
		return &pbPath.PathResponse{Found: false, ErrorMessage: "No path found"}, nil
	}

	return &pbPath.PathResponse{Found: true, PathNodeIds: path}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":"+PATHFINDER_PORT)
	if err != nil {
		log.Fatalf("Fallo al escuchar puerto %s: %v", PATHFINDER_PORT, err)
	}

	s := &PathfinderServer{}
	grpcServer := grpc.NewServer()

	pbPath.RegisterPathfinderServiceServer(grpcServer, s)

	go s.syncWithSimulation()

	log.Printf("🧭 Pathfinder Service corriendo en puerto %s", PATHFINDER_PORT)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Fallo al servir gRPC: %v", err)
	}
}

func (s *PathfinderServer) syncWithSimulation() {
	target := config.GetAddress()

	for {
		log.Printf("Conectando a Simulación en %s...", target)
		conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Error conectando: %v. Reintentando en 2s...", err)
			time.Sleep(2 * time.Second)
			continue
		}

		client := pbSim.NewSimulationServiceClient(conn)

		stream, err := client.StreamSimulation(context.Background(), &pbSim.Empty{})
		if err != nil {
			log.Printf("Error abriendo stream: %v. Reintentando...", err)
			conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		log.Println("✅ Conectado a Simulación. Sincronizando grafo...")

		for {
			state, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Error recibiendo datos: %v", err)
				break
			}

			newGraph := model.BuildFromState(state.Nodes)

			s.mu.Lock()
			s.latestGraph = newGraph
			s.mu.Unlock()
		}

		conn.Close()
		time.Sleep(1 * time.Second)
	}
}
