package model

import (
	"math"
)

const (
	EARTH_MASS              = 5.972e+24
	EARTH_RADIUS            = 6.371e+6
	EARTH_ROTATION_CYCLE    = 86400
	SIMULATION_SPEED        = 1440
	SIMULATION_SCALE_TO_ONE = EARTH_RADIUS
	G                       = 6.67430e-11
	ID_HASH_LEN             = 5
	TICK_DELAY              = 10
)

var (
	EARTH_GEOSTATIONARY_ALTITUDE = math.Cbrt((G*EARTH_MASS*math.Pow(EARTH_ROTATION_CYCLE, 2))/(4*math.Pow(math.Pi, 2))) - EARTH_RADIUS
	MIN_TRIANGULATION_ALTITUDE   = EARTH_RADIUS
)
