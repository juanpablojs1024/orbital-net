package main

import (
	"orbital-net/services/sim-service/internal/model"
	"orbital-net/services/sim-service/internal/server"
	"time"
)

func main() {
	model.CreateSim()
	go runSimulationLoop()
	server.StartGRPCServer()
}

func runSimulationLoop() {
	ticker := time.NewTicker(model.TICK_DELAY * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		model.SimMutex.Lock()

		model.TickSimulation()
		model.GetVisibility()

		model.SimMutex.Unlock()
	}
}
