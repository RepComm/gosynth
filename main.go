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
	go se.Synth()

	HandleSigTerm(func(sig os.Signal) {
		os.Exit(0)
	})

	for {
		time.Sleep(10 * time.Second)
	}
}

type SynthEngine struct {
	SampleRate    float32
	GlobalTime    float32
	OutputChannel *RingBuffer
	OutputReader  *RingReader
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
	}, nil
}

const SYNTH_BLOCK_SIZE = 512

func (se *SynthEngine) Generate(block []float32) int {

	phase := se.GlobalTime
	var sinFreq float32 = 440.0
	var sinAmp float32 = 0.5

	phaseStep := float32(2*math.Pi) * sinFreq / se.SampleRate

	blockSize := len(block)
	for i := 0; i < blockSize; i++ {
		block[i] = Sin32(phase) * sinAmp
		phase += phaseStep
	}

	se.GlobalTime = phase
	return blockSize
}

func (se *SynthEngine) Synth() {

	var block [SYNTH_BLOCK_SIZE]float32

	i := 0
	for {
		se.Generate(block[:])
		se.OutputChannel.Write(block[:])
		i++
		// if n < 1 {
		// 	break
		// }
	}
	// fmt.Println("wrote", i)

}

func Sin32(f float32) float32 {
	return float32(math.Sin(float64(f)))
}
