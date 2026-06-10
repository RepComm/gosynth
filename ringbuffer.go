package main

import (
	"encoding/binary"
	"errors"
	"math"
	"sync/atomic"
)

type RingBuffer struct {
	Buffer     []float32
	BufferSize uint64
	//used to convert WritePos and ReadPos to buffer indices
	BufferMask uint64

	WritePos atomic.Uint64
	ReadPos  atomic.Uint64
}

const RING_BUFFER_DEFAULT_SIZE uint64 = 2048

func IsPowerOfTwo(i int) bool {
	return i > 0 && (i&(i-1)) == 0
}

func NewRingBuffer(size uint64) (*RingBuffer, error) {
	if !IsPowerOfTwo(int(size)) {
		return nil, errors.New("ring buffer size is not a power of two, but relies on powers of two for fast operations.")
	}
	result := &RingBuffer{
		BufferSize: size,
		Buffer:     make([]float32, size),
		BufferMask: size - 1,
	}
	return result, nil
}

func (rb *RingBuffer) Write(data []float32) int {
	writePos := rb.WritePos.Load()
	readPos := rb.ReadPos.Load()
	available := writePos - readPos

	//clamp buffer size
	if available > rb.BufferSize {
		available = rb.BufferSize
	}
	free := rb.BufferSize - available

	toWrite := min(
		uint64(len(data)),
		free,
	)

	writeIndex := writePos & rb.BufferMask

	first := min(
		toWrite,
		rb.BufferSize-writeIndex,
	)

	total := copy(
		rb.Buffer[writeIndex:],
		data[:first],
	)

	total += copy(
		rb.Buffer,
		data[first:toWrite],
	)

	rb.WritePos.Store(writePos + uint64(total))

	return total
}

// available for consumption
func (rb *RingBuffer) Available() uint64 {
	return rb.WritePos.Load() - rb.ReadPos.Load()
}

// available for writing
func (rb *RingBuffer) Free() uint64 {
	return rb.BufferSize - rb.Available()
}

func (rb *RingBuffer) Read(out []float32) int {

	read := rb.ReadPos.Load()
	write := rb.WritePos.Load()

	available := write - read

	toRead := min(
		uint64(len(out)),
		available,
	)

	readIndex := read & rb.BufferMask

	first := min(
		toRead,
		rb.BufferSize-readIndex,
	)

	total := copy(
		out,
		rb.Buffer[readIndex:readIndex+first],
	)

	total += copy(
		out[first:],
		rb.Buffer[:toRead-first],
	)

	rb.ReadPos.Store(read + uint64(total))

	return total
}

const RING_READER_SCRATCH_BUFFER_SIZE = 256

type RingReader struct {
	RB      *RingBuffer
	Scratch [RING_READER_SCRATCH_BUFFER_SIZE]float32
}

func (rr *RingReader) Read(output []byte) (int, error) {
	//output size wanted in bytes
	outputLen := len(output)
	//output amount in samples
	outputSampleLen := outputLen / 4

	//assume we're going to provide all the samples asked for
	readSampleCount := outputSampleLen

	scratchLen := len(rr.Scratch)
	//make sure we don't read overflow the scratch buffer
	if readSampleCount > scratchLen {
		readSampleCount = scratchLen
	}

	//read the data from the ring buffer
	n := rr.RB.Read(rr.Scratch[:readSampleCount])

	//iterate over samples read from ring buffer
	for i := 0; i < n; i++ {
		//get the sample
		sample := rr.Scratch[i]

		//convert to bits
		bits := math.Float32bits(sample)

		outputStartIndex := i * 4
		outputEndIndex := outputStartIndex + 4
		binary.LittleEndian.PutUint32(
			output[outputStartIndex:outputEndIndex],
			bits,
		)
	}

	return n * 4, nil
}
