package model

import (
	"math"
	"sync"
)

var (
	ParentPlanet     *Planet
	Nodes            []*Node
	VisibilityMatrix [][]bool
	SimMutex         sync.RWMutex
)

func createVisibilityMatrix(n int) {
	VisibilityMatrix = make([][]bool, n)
	for i := 0; i < n; i++ {
		VisibilityMatrix[i] = make([]bool, n)
	}
}

func GetVisibility() {
	n := len(VisibilityMatrix)
	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			VisibilityMatrix[i][j] = Nodes[i].CanView(Nodes[j])
			VisibilityMatrix[j][i] = VisibilityMatrix[i][j]
		}
	}
}

func TickSimulation() {
	for i := range Nodes {
		Nodes[i].Move()
	}
}

func AddNode(name string, altitude, angle float64, isStation bool, portQty, portGen int) string {
	SimMutex.Lock()
	defer SimMutex.Unlock()

	if portQty <= 0 {
		portQty = 4
	}
	if portGen <= 0 {
		portGen = 4
	}

	var newNode *Node

	if isStation {
		newNode = NewGroundNode(name, ParentPlanet, angle, portQty, portGen)
	} else {
		newNode = NewOrbitNode(name, ParentPlanet, altitude, angle, portQty, portGen)
	}

	Nodes = append(Nodes, newNode)
	createVisibilityMatrix(len(Nodes))

	return newNode.ID
}

func RemoveNode(id string) bool {
	SimMutex.Lock()
	defer SimMutex.Unlock()

	index := -1
	for i, n := range Nodes {
		if n.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return false
	}

	Nodes = append(Nodes[:index], Nodes[index+1:]...)

	createVisibilityMatrix(len(Nodes))
	return true
}

func ClearAll() {
	SimMutex.Lock()
	defer SimMutex.Unlock()

	Nodes = []*Node{}
	createVisibilityMatrix(0)
}

func CreateSim() {
	ParentPlanet = NewPlanet("Earth", EARTH_RADIUS, EARTH_MASS, EARTH_ROTATION_CYCLE)
	Nodes = []*Node{
		NewOrbitNode("Gio", ParentPlanet, MIN_TRIANGULATION_ALTITUDE+1000, 1*math.Pi*2/3, 1, 2),
		NewOrbitNode("Timo-T", ParentPlanet, MIN_TRIANGULATION_ALTITUDE+1000, 2*math.Pi*2/3, 1, 2),
		NewOrbitNode("Gonzalito", ParentPlanet, MIN_TRIANGULATION_ALTITUDE+1000, 3*math.Pi*2/3, 1, 2),

		NewOrbitNode("Lola", ParentPlanet, 10000000, 1*math.Pi*2/4, 1, 2),
		NewOrbitNode("Martina", ParentPlanet, 10000000, 2*math.Pi*2/4, 1, 2),
		NewOrbitNode("Tulio", ParentPlanet, 10000000, 3*math.Pi*2/4, 1, 2),
		NewOrbitNode("Silvio", ParentPlanet, 10000000, 4*math.Pi*2/4, 1, 2),

		NewGroundNode("Honey", ParentPlanet, 1*math.Pi*2/3, 4, 6),
		NewGroundNode("Bonnie", ParentPlanet, 2*math.Pi*2/3, 4, 6),
		NewGroundNode("Kissie", ParentPlanet, 3*math.Pi*2/3, 4, 6),
	}
	createVisibilityMatrix(len(Nodes))
}
