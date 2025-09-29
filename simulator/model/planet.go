package model

type Planet struct {
	Name       string
	Radius     float64
	ThetaSpeed float64
}

func NewPlanet(name string, radius, thetaSpeed float64) *Planet {
	return &Planet{Name: name, Radius: radius, ThetaSpeed: thetaSpeed}
}
