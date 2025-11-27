package server

import (
	"context"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"

	"orbital-net/pkg/config"
	pbCommon "orbital-net/pkg/proto/common"
	pbSim "orbital-net/pkg/proto/sim"
	"orbital-net/services/sim-service/internal/model"
)

type SimServer struct {
	pbSim.UnimplementedSimulationServiceServer
}

func (s *SimServer) StreamSimulation(_ *pbSim.Empty, stream pbSim.SimulationService_StreamSimulationServer) error {
	log.Println(">>> Nuevo Cliente conectado")
	defer log.Println("<<< Cliente desconectado")

	ticker := time.NewTicker(config.SERVER_TICK_DELAY * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			log.Println("--- El cliente cerró la conexión (Context Done)")
			return nil
		case <-ticker.C:
			model.SimMutex.RLock()

			var protoNodes []*pbCommon.Node

			for i, n := range model.Nodes {
				var visibleIDs []string
				for j, target := range model.Nodes {
					if i != j && model.VisibilityMatrix[i][j] {
						visibleIDs = append(visibleIDs, target.ID)
					}
				}

				x, y := n.XYPosition()
				alt := n.Orbit.Radius - n.ParentPlanet.Radius

				protoNodes = append(protoNodes, &pbCommon.Node{
					Id:           n.ID,
					Name:         n.Name,
					X:            x,
					Y:            y,
					Speed:        n.GetLinearSpeed(),
					Altitude:     alt,
					Angle:        n.Orbit.ThetaPos,
					VisiblePeers: visibleIDs,
					PortQuantity: int32(n.Interface.PortQuantity),
				})
			}

			model.SimMutex.RUnlock()

			resp := &pbSim.SimulationState{
				Timestamp: time.Now().UnixMilli(),
				Nodes:     protoNodes,
			}

			if err := stream.Send(resp); err != nil {
				log.Printf("Error enviando stream: %v", err)
				return err
			}
		}
	}
}

func StartGRPCServer() {
	lis, err := net.Listen("tcp", ":"+config.GRPC_PORT)
	if err != nil {
		log.Fatalf("Fallo al escuchar puerto %s: %v", config.GRPC_PORT, err)
	}

	grpcServer := grpc.NewServer()
	pbSim.RegisterSimulationServiceServer(grpcServer, &SimServer{})

	log.Printf("Simulación corriendo. Servidor gRPC escuchando en %s", config.GRPC_PORT)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Fallo al servir gRPC: %v", err)
	}
}

func (s *SimServer) CreateNode(ctx context.Context, req *pbSim.CreateNodeRequest) (*pbSim.CreateNodeResponse, error) {
	isStation := (req.Type == pbSim.NodeType_STATION)

	id := model.AddNode(req.Name, req.Altitude, req.Angle, isStation, int(req.PortQuantity), int(req.PortGeneration))

	log.Printf("🏗️ Nuevo nodo lanzado: %s (%s) [Station: %v, Ports: %d, Gen: %d]",
		req.Name, id, isStation, req.PortQuantity, req.PortGeneration)

	return &pbSim.CreateNodeResponse{NodeId: id, Success: true}, nil
}

func (s *SimServer) DeleteNode(ctx context.Context, req *pbSim.DeleteNodeRequest) (*pbSim.DeleteNodeResponse, error) {
	success := model.RemoveNode(req.NodeId)
	if success {
		log.Printf("🗑️ Nodo eliminado: %s", req.NodeId)
	}
	return &pbSim.DeleteNodeResponse{Success: success}, nil
}

func (s *SimServer) ClearSimulation(ctx context.Context, _ *pbSim.Empty) (*pbSim.Empty, error) {
	model.ClearAll()
	log.Println("🔥 ¡Simulación reiniciada (Clear All)!")
	return &pbSim.Empty{}, nil
}
