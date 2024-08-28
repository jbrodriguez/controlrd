package sensor

import "github.com/jbrodriguez/controlrd/daemon/dto"

// NoSensor -
type NoSensor struct {
	samples []dto.Sample
}

// NewNoSensor -
func NewNoSensor() *NoSensor {
	nosensor := &NoSensor{
		samples: make([]dto.Sample, 0),
	}
	return nosensor
}

// GetReadings -
func (n *NoSensor) GetReadings(_ dto.Prefs) []dto.Sample {
	return n.samples
}
