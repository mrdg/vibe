package main

import (
	"io"
	"math"
	"os"
	"time"

	wav "github.com/youpy/go-wav"
)

type timeSig struct {
	num   int
	denom int
}

// clock keeps track of the number of samples seen since starting the audio thread
type clock struct {
	sampleRate float64

	samples  uint64 // total time passed in number of samples
	nextTick uint64 // time of next tick in number of samples
}

// tick is called once for every audio callback. If the next tick falls within the current
// audio buffer, its offset in the buffer is returned. Otherwise it returns -1
func (c *clock) tick(state state) int {
	// BPM is assumed to be specified as a quarter note value, i.e. ♩ = 120,
	// regardless of the time signature. This seems to be what DAWs are doing
	// as well.
	var (
		sig          = state.timeSig
		ticksPerBeat = (state.patternLen / sig.num) * (sig.denom / 4)
		ticksPerSec  = (state.bpm * float64(ticksPerBeat)) / 60.0
		tickDuration = uint64(c.sampleRate / ticksPerSec)
	)

	frame := int(c.nextTick - c.samples)
	c.samples += uint64(state.bufferSize)

	if frame < state.bufferSize {
		c.nextTick += tickDuration
		return frame
	}
	return -1
}

type machine struct {
	clock  *clock
	sounds []*sound
	sum    []float64
}

func (m *machine) process(state state, out []int16) {
	tick := m.clock.tick(state)

	for i, snd := range m.sounds {
		gain := math.Pow(10, state.gain[i]/20.0)
		env := envDecay(m.clock.sampleRate, state.decay[i])

		// continue outputting samples for voices already in progress
		for k, pos := range snd.voices {
			if pos > 0 {
				snd.voices[k] = sum(m.sum[0:], snd.buf, pos, gain, env)
			}
		}
		// check if a new sample should start in the current audio buffer
		if tick == -1 {
			continue
		}
		pattern := state.patterns[i]
		step := &state.steps[i]
		if *step >= len(pattern) {
			*step = 0
		}
		if pattern[*step] != 0 && !state.muted[i] {
			for k, pos := range snd.voices {
				if pos == 0 {
					// multiply tick by 2 because sample buffer is stereo-interleaved
					snd.voices[k] = sum(m.sum[tick*2:], snd.buf, 0, gain, env)
					break
				}
			}
		}
		*step++
	}

	const scale = 1 << 15 // assumes 16 bit output

	// write samples to output buffer
	for i, sample := range m.sum {
		out[i] = int16(scale * sample)
		m.sum[i] = 0.0
	}
}

func envDecay(sampleRate float64, decay time.Duration) func(int) float64 {
	decaySamples := sampleRate * (float64(decay.Microseconds()) / float64(time.Second.Microseconds()))
	return func(pos int) float64 {
		if float64(pos) > decaySamples {
			return 0
		}
		return float64(-pos)*(1.0/decaySamples) + 1
	}
}

// sum adds samples from src to dst, starting at offset, and returns
// the new src offset.
func sum(dst, src []float64, offset int, gain float64, env func(pos int) float64) int {
	n := min(len(src)-offset, len(dst))
	for i, sample := range src[offset : offset+n] {
		dst[i] += sample * gain * env(offset+i)
	}
	offset += n
	if offset >= len(src) {
		offset = 0
	}
	return offset
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

const maxVoices = 12

type sound struct {
	buf []float64

	// voices keeps track of positions in buf to allow overlapping instances of
	// the same sound. When 0, the voice is unused.
	voices [maxVoices]int
}

func loadSound(path string) (*sound, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := wav.NewReader(f)
	var snd sound
	for {
		samples, err := r.ReadSamples()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		for _, sample := range samples {
			v := r.FloatValue(sample, 0)
			snd.buf = append(snd.buf, v) // L
			snd.buf = append(snd.buf, v) // R
		}
	}
	return &snd, nil
}

func mustLoadSound(path string) *sound {
	snd, err := loadSound(path)
	if err != nil {
		panic(err)
	}
	return snd
}
