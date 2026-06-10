package main

import (
	"math"
	"os"
	"slices"
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
	OutputSample  *float64
}

func NewSynthEngine() (*SynthEngine, error) {
	rb, err := NewRingBuffer(RING_BUFFER_DEFAULT_SIZE)
	if err != nil {
		return nil, err
	}
	var outputSample float64 = 0
	result := &SynthEngine{
		SampleRate:    48000.0,
		GlobalTime:    0,
		OutputChannel: rb,
		OutputReader:  &RingReader{RB: rb},
		Oscs:          []*Osc{},
		OutputSample:  &outputSample,
	}

	a := result.AddOsc(&Osc{
		Type:      OSC_TYPE_SIN,
		Frequency: 440.0,
		Amplitude: 0.5,
	})

	result.AddOsc(&Osc{
		Type:      OSC_TYPE_SIN,
		Frequency: 2.0,
		Amplitude: 0.01,
		Output:    &a.Frequency,
	})

	return result, nil
}
func (se *SynthEngine) AddOsc(osc *Osc) *Osc {
	if slices.Contains(se.Oscs, osc) {
		return osc
	}
	if osc.Output == nil {
		osc.Output = se.OutputSample
	}
	se.Oscs = append(se.Oscs, osc)
	return osc
}

const SYNTH_BLOCK_SIZE = 1024
const SYNTH_BLOCK_LOW = 256
const SYNTH_WORKER_SLEEP_MS = 1

const Tau float64 = math.Pi * 2

func (se *SynthEngine) Generate(block []float32) int {

	phase := se.GlobalTime

	blockSize := len(block)

	for i := 0; i < blockSize; i++ {
		*se.OutputSample = 0
		for _, osc := range se.Oscs {
			osc.Sample64(se.SampleRate, true, osc.Output)
		}

		block[i] = float32(*se.OutputSample)
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
