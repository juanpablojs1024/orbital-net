package simulation

import (
	"math"
	"sync"

	"satellite-coms/simulator/model"
)

var (
	planet *model.Planet
	Nodes  []*model.Node
	Mutex  sync.Mutex
)

func InitSimulation() {
	planet = model.NewPlanet("Earth", 1, math.Pi/20480)

	Nodes = []*model.Node{
		model.NewSatellite("Gonzalito", planet, math.Sqrt(2), 3*(math.Pi*2/3), math.Pi/5120, 1, 6),
		model.NewSatellite("Giovanni", planet, math.Sqrt(2), 2*(math.Pi*2/3), math.Pi/5120, 1, 6),
		model.NewSatellite("Martina", planet, math.Sqrt(2), 1*(math.Pi*2/3), math.Pi/5120, 1, 6),

		model.NewSatellite("Bonnie", planet, 3, 3*(math.Pi*2/3), math.Pi/10240, 3, 3),
		model.NewSatellite("Kissie", planet, 3, 2*(math.Pi*2/3), math.Pi/10240, 3, 3),
		model.NewSatellite("Honey", planet, 3, 1*(math.Pi*2/3), math.Pi/10240, 3, 3),
		model.NewServer("Home", planet, 0, 2, 2),
		model.NewServer("Office", planet, math.Pi, 6, 7)}
}
