package model

import (
	"crypto/sha1"
	"encoding/hex"
	"math"
)

type Node struct {
	ID           string
	Name         string
	ParentPlanet *Planet
	OrbitRadius  float64
	OrbitTheta   float64
	ThetaSpeed   float64
	Ports        int
	PortGen      int
}

func NewSatellite(name string, parentPlanet *Planet, orbitRadius, orbitTheta, thetaSpeed float64, ports int, portGen int) *Node {
	return &Node{ID: "sat_" + hashID(name)[:5], Name: name, ParentPlanet: parentPlanet, OrbitRadius: orbitRadius, OrbitTheta: orbitTheta, ThetaSpeed: thetaSpeed, Ports: ports, PortGen: portGen}
}

func NewServer(name string, parentPlanet *Planet, positionTheta float64, ports int, portGen int) *Node {
	return &Node{ID: "srv_" + hashID(name)[:5], Name: name, ParentPlanet: parentPlanet, OrbitRadius: parentPlanet.Radius, OrbitTheta: positionTheta, ThetaSpeed: parentPlanet.ThetaSpeed, Ports: ports, PortGen: portGen}
}

func (n *Node) Position() (float64, float64) {
	return n.OrbitRadius * math.Cos(n.OrbitTheta), n.OrbitRadius * math.Sin(n.OrbitTheta)
}

func (n *Node) Move() {
	n.OrbitTheta += n.ThetaSpeed
}

func (n1 *Node) CanView(n2 *Node) bool {
	x1, y1 := n1.Position()
	x2, y2 := n2.Position()

	A := x2 - x1
	B := y2 - y1
	C := x2*y1 - x1*y2
	D := math.Abs(C) / math.Sqrt(A*A+B*B)
	T := (A*(-x1) + B*(-y1)) / (A*A + B*B)

	return D >= n1.ParentPlanet.Radius || T < 0 || T > n1.ParentPlanet.Radius
}

func hashID(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
