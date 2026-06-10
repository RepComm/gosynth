package main

import (
	"math"
	"os"
	"time"

	"github.com/ebitengine/oto/v3"
)

func main() {
	//create our synth engine
	se, err := NewSynthEngine()
	if err != nil {
		panic(err)
	}

	//setup oto audio driver
	op := &oto.NewContextOptions{
		SampleRate:   48000,
		ChannelCount: 1,
		Format:       oto.FormatFloat32LE,
	}

	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		panic(err)
	}

	//wait for driver ready
	<-ready

	//connect driver to synth engine output
	otoPlayer := ctx.NewPlayer(se.OutputReader)
	otoPlayer.Play()

	//start our synth engine
	go se.Worker()

	HandleSigTerm(func(sig os.Signal) {
		os.Exit(0)
	})

	for {
		time.Sleep(1 * time.Second)
	}
}

type SynthEngine struct {
	SampleRate    float64
	GlobalTime    float64
	OutputChannel *RingBuffer
	OutputReader  *RingReader
	Oscs          []*Osc
}

func NewSynthEngine() (*SynthEngine, error) {
	rb, err := NewRingBuffer(RING_BUFFER_DEFAULT_SIZE)
	if err != nil {
		return nil, err
	}
	return &SynthEngine{
		SampleRate:    48000.0,
		GlobalTime:    0,
		OutputChannel: rb,
		OutputReader:  &RingReader{RB: rb},
		Oscs: []*Osc{
			&Osc{
				Type:      OSC_TYPE_SIN,
				Frequency: 440.0,
				Amplitude: 0.5,
			},
			&Osc{
				Type:      OSC_TYPE_SAW,
				Frequency: 440.0,
				Amplitude: 0.1,
			},
		},
	}, nil
}

const SYNTH_BLOCK_SIZE = 1024
const SYNTH_BLOCK_LOW = 256
const SYNTH_WORKER_SLEEP_MS = 5

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
}

const Tau float64 = math.Pi * 2

func (osc *Osc) Sample64(sampleRate float64) float64 {

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

	osc.Phase += Tau * osc.Frequency / sampleRate
	if osc.Phase >= Tau {
		osc.Phase -= Tau
	}

	return sample * osc.Amplitude
}

func (se *SynthEngine) Generate(block []float32) int {

	phase := se.GlobalTime

	blockSize := len(block)
	var sample64 float64
	for i := 0; i < blockSize; i++ {
		sample64 = 0

		for _, osc := range se.Oscs {
			sample64 += osc.Sample64(se.SampleRate)
		}

		block[i] = float32(sample64)
	}

	se.GlobalTime = phase
	return blockSize
}

func (se *SynthEngine) Worker() {

	var block [SYNTH_BLOCK_SIZE]float32

	for {
		available := se.OutputChannel.Available()
		// fmt.Printf(
		// 	"avail=%d free=%d write=%d read=%d\n",
		// 	se.OutputChannel.Available(),
		// 	se.OutputChannel.Free(),
		// 	se.OutputChannel.WritePos.Load(),
		// 	se.OutputChannel.ReadPos.Load(),
		// )
		if available < SYNTH_BLOCK_LOW {
			se.Generate(block[:])
			se.OutputChannel.Write(block[:])
		} else {
			time.Sleep(SYNTH_WORKER_SLEEP_MS * time.Millisecond)
		}
	}
}
