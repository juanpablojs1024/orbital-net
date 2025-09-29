package handler

import (
	"encoding/json"
	"net/http"
	"satellite-coms/simulator/simulation"
)

func GetPositionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	simulation.Mutex.Lock()
	defer simulation.Mutex.Unlock()

	positions := make([]map[string]interface{}, len(simulation.Nodes))
	for i, node := range simulation.Nodes {
		x, y := node.Position()
		positions[i] = map[string]interface{}{
			"id":      node.ID,
			"name":    node.Name,
			"x":       x,
			"y":       y,
			"ports":   node.Ports,
			"portgen": node.PortGen,
		}
	}
	json.NewEncoder(w).Encode(positions)
}

func GetVisibilityMatrixHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	simulation.Mutex.Lock()
	defer simulation.Mutex.Unlock()

	matrix := make([][]bool, len(simulation.Nodes))
	for i := range matrix {
		matrix[i] = make([]bool, len(simulation.Nodes))
		for j := range matrix[i] {
			if i != j {
				matrix[i][j] = simulation.Nodes[i].CanView(simulation.Nodes[j])
			}
		}
	}
	json.NewEncoder(w).Encode(matrix)
}

func StepHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	simulation.Mutex.Lock()
	defer simulation.Mutex.Unlock()

	for _, node := range simulation.Nodes {
		node.Move()
	}
	w.Write([]byte("OK"))
}
