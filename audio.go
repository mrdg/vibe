package main

import (
	"io"
	"math"
	"os"

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
func (c *clock) tick(state state) (int, bool) {
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
		return frame, true
	}
	return frame, false
}

type machine struct {
	clock *clock
	sum   []float64
}

const chokeDecay = 0.05 // 50ms

func (m *machine) process(state *state, out []float32) {
	offset, tick := m.clock.tick(*state)

	for _, snd := range state.sounds {
		if state.step >= state.patternLen {
			state.step = 0
		}
		gain := math.Pow(10, snd.gain/20.0)

		choked := false
		if tick {
			for _, other := range snd.chokeGroup {
				if other.pattern[state.step] != 0 {
					choked = true
				}
			}
		}

		// continue to output samples for active voices
		for _, voice := range snd.voices {
			if voice.pos > 0 {
				if choked && !voice.choked {
					// set envelope to short decay to choke sound
					voice.env.startSample = voice.pos + offset*2
					voice.env.decaySamples = m.clock.sampleRate * chokeDecay
					voice.choked = true
				}
				voice.pos = sum(m.sum[0:], snd.buf, voice.pos, gain, voice.env.value)
			}
		}

		// trigger a new voice
		if tick && snd.pattern[state.step] != 0 && !snd.muted {
			voice := snd.findFreeVoice()
			voice.env.startSample = 0
			voice.env.decaySamples = m.clock.sampleRate * snd.decay
			voice.choked = false
			voice.pos = sum(m.sum[offset*2:], snd.buf, 0, gain, voice.env.value)
		}
	}

	if tick {
		state.step++
		if state.step >= state.patternLen {
			state.step = 0
		}
	}

	for i := range m.hits {
		m.hits[i] = 0
	}

	// write samples to output buffer
	for i, sample := range m.sum {
		out[i] = float32(sample)
		m.sum[i] = 0.0
	}
}

type envelope struct {
	startSample  int
	decaySamples float64
}

func (e *envelope) value(pos int) float64 {
	if e.startSample == -1 || pos < e.startSample {
		return 1.0
	}
	if float64(pos) > float64(e.startSample)+e.decaySamples {
		return 0
	}
	start := float64(pos - e.startSample)
	return -start*(1.0/e.decaySamples) + 1
}

// sum adds samples from src to dst, starting at offset, and returns
// the new src offset.
func sum(dst, src []float64, offset int, gain float64, env func(int) float64) int {
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
	file string
	buf  []float64

	// voices allow multiple instances of the same sound to overlap
	voices []*voice

	pattern []int
	probs   []float64
	muted   bool
	gain    float64 // in dB
	decay   float64 // seconds

	chokeGroup []*sound
}

func (s sound) findFreeVoice() *voice {
	for _, voice := range s.voices {
		if voice.pos == 0 {
			return voice
		}
	}
	panic("no free voice found")
}

type voice struct {
	pos    int
	env    envelope
	choked bool
}

func loadSound(path string, patternLen int) (*sound, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := wav.NewReader(f)
	snd := sound{
		file:    path,
		decay:   2,
		gain:    1.,
		pattern: make([]int, patternLen),
		probs:   make([]float64, patternLen),
	}
	for i := range snd.probs {
		snd.probs[i] = 1.
	}
	for i := 0; i < maxVoices; i++ {
		snd.voices = append(snd.voices, &voice{})
	}
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

func mustLoadSound(path string, patternLen int) *sound {
	snd, err := loadSound(path, patternLen)
	if err != nil {
		panic(err)
	}
	return snd
}
