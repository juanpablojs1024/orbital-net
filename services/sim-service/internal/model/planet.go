package model

import "math"

type Planet struct {
	Name               string
	Radius             float64
	Mass               float64
	RotationThetaSpeed float64
}

func NewPlanet(name string, radius, mass, rotationCycle float64) *Planet {
	return &Planet{Name: name, Radius: radius, Mass: mass, RotationThetaSpeed: 2 * math.Pi / rotationCycle}
}
