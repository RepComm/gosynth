package main

import "math"

type OscType int

const (
	OSC_TYPE_SIN OscType = iota
	OSC_TYPE_COS
	OSC_TYPE_SAW
	OSC_TYPE_SQUARE
	OSC_TYPE_TRIANGLE
)

type Osc struct {
	Frequency float64
	Amplitude float64
	Phase     float64
	Type      OscType
	Output    *float64
}

func (osc *Osc) Sample64(sampleRate float64, Advance bool, output *float64) {

	var sample float64

	switch osc.Type {
	case OSC_TYPE_SIN:
		sample = math.Sin(osc.Phase)

	case OSC_TYPE_SAW:
		sample = osc.Phase/math.Pi - 1

	case OSC_TYPE_SQUARE:
		if osc.Phase < math.Pi {
			sample = 1
		} else {
			sample = -1
		}
	}

	if Advance {
		osc.Phase += Tau * osc.Frequency / sampleRate
		if osc.Phase >= Tau {
			osc.Phase -= Tau
		}
	}

	*output += sample * osc.Amplitude
}
