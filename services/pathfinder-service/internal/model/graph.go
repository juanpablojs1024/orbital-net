package model

import (
	pbCommon "orbital-net/pkg/proto/common"
)

type Graph struct {
	Adj     map[string][]string
	PortGen map[string]int32
}

func NewGraph() *Graph {
	return &Graph{
		Adj:     make(map[string][]string),
		PortGen: make(map[string]int32),
	}
}

func BuildFromState(nodes []*pbCommon.Node) *Graph {
	g := NewGraph()

	for _, n := range nodes {
		gen := n.PortGeneration
		if gen <= 0 {
			gen = 1
		}
		g.PortGen[n.Id] = gen

		if _, exists := g.Adj[n.Id]; !exists {
			g.Adj[n.Id] = []string{}
		}

		for _, neighborID := range n.VisiblePeers {
			g.Adj[n.Id] = append(g.Adj[n.Id], neighborID)
		}
	}
	return g
}

func (g *Graph) FindWidestPath(startID, endID string) ([]string, bool) {
	if startID == endID {
		return []string{startID}, true
	}

	type state struct {
		node  string
		minPG int32
	}

	maxPortgenSoFar := make(map[string]int32)
	prev := make(map[string]string)
	visited := make(map[string]bool)

	startGen := g.PortGen[startID]
	queue := []state{{node: startID, minPG: startGen}}
	maxPortgenSoFar[startID] = startGen

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

		if current.node == endID {
			path := []string{endID}
			for node := endID; node != startID; {
				node = prev[node]
				path = append([]string{node}, path...)
			}
			return path, true
		}

		for _, neighbor := range g.Adj[current.node] {
			if visited[neighbor] {
				continue
			}

			neighborGen := g.PortGen[neighbor]
			newMinPG := min(current.minPG, neighborGen)

			if newMinPG > maxPortgenSoFar[neighbor] {
				maxPortgenSoFar[neighbor] = newMinPG
				prev[neighbor] = current.node
				queue = append(queue, state{node: neighbor, minPG: newMinPG})
			}
		}
	}

	return nil, false
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func (g *Graph) FindShortestPath(startID, endID string) ([]string, bool) {
	return g.FindWidestPath(startID, endID)
}
