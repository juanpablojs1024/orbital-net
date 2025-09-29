package model

import (
	"fmt"
	"log"

	"satellite-coms/pathfinder/internal/httpclient"
)

type Graph struct {
	adj     map[string][]string
	portgen map[string]int
}

func NewGraph() *Graph {
	return &Graph{
		adj:     make(map[string][]string),
		portgen: make(map[string]int),
	}
}

func (g *Graph) AddEdges(node string, neighbors []string) {
	for _, neighbor := range neighbors {
		if !contains(g.adj[node], neighbor) {
			g.adj[node] = append(g.adj[node], neighbor)
		}
		if !contains(g.adj[neighbor], node) {
			g.adj[neighbor] = append(g.adj[neighbor], node)
		}
	}
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func (g *Graph) Print() {
	for node, neighbors := range g.adj {
		fmt.Printf("%s (portgen: %d) -> %v\n", node, g.portgen[node], neighbors)
	}
}

func (g *Graph) WidestPath(start, end string, restricted map[string]bool) ([]string, bool) {
	if start == end {
		return []string{start}, true
	}

	type state struct {
		node  string
		minPG int
	}

	maxPortgen := map[string]int{}
	prev := map[string]string{}
	visited := map[string]bool{}
	queue := []state{{node: start, minPG: g.portgen[start]}}
	maxPortgen[start] = g.portgen[start]

	for len(queue) > 0 {
		bestIdx := 0
		for i := 1; i < len(queue); i++ {
			if queue[i].minPG > queue[bestIdx].minPG {
				bestIdx = i
			}
		}
		current := queue[bestIdx]
		queue = append(queue[:bestIdx], queue[bestIdx+1:]...)

		if visited[current.node] {
			continue
		}
		visited[current.node] = true

		if current.node == end {
			path := []string{end}
			for node := end; node != start; {
				node = prev[node]
				path = append([]string{node}, path...)
			}
			return path, true
		}

		for _, neighbor := range g.adj[current.node] {
			if neighbor != start && neighbor != end && restricted[neighbor] {
				continue
			}
			if visited[neighbor] {
				continue
			}

			edgePG := min(g.portgen[current.node], g.portgen[neighbor])
			newMin := min(current.minPG, edgePG)

			if newMin > maxPortgen[neighbor] {
				maxPortgen[neighbor] = newMin
				prev[neighbor] = current.node
				queue = append(queue, state{node: neighbor, minPG: newMin})
			}
		}
	}
	return nil, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- Build the graph using Redis + Consul services ---
func CreateGraph() *Graph {
	g := NewGraph()

	// Fetch nodes and visibility from the simulator service
	nodes := httpclient.FetchNodes()
	matrix := httpclient.FetchVisibility()

	for idx, node := range nodes {
		obj := node.(map[string]interface{})
		id := obj["id"].(string)

		// Store portgen for the base node
		if portgenVal, ok := obj["portgen"].(float64); ok {
			g.portgen[id] = int(portgenVal)
		} else {
			g.portgen[id] = -1
		}

		// Number of ports per node
		ports := 1
		if portsVal, ok := obj["ports"].(float64); ok && int(portsVal) > 0 {
			ports = int(portsVal)
		}

		row, ok := matrix[idx].([]interface{})
		if !ok {
			log.Fatalf("Row %d is not a []interface{}", idx)
		}

		// Create port nodes and connect edges
		for portNum := 1; portNum <= ports; portNum++ {
			nodePort := fmt.Sprintf("%s:port%d", id, portNum)

			var neighbors []string
			for j, val := range row {
				if val == true {
					neighborObj := nodes[j].(map[string]interface{})
					neighborID := neighborObj["id"].(string)
					neighborPorts := 1
					if npVal, ok := neighborObj["ports"].(float64); ok && int(npVal) > 0 {
						neighborPorts = int(npVal)
					}
					for np := 1; np <= neighborPorts; np++ {
						neighborPort := fmt.Sprintf("%s:port%d", neighborID, np)
						neighbors = append(neighbors, neighborPort)
					}
				}
			}

			g.AddEdges(nodePort, neighbors)
			g.portgen[nodePort] = g.portgen[id]
		}
	}

	return g
}
