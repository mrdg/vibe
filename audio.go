package main

import (
	"fmt"
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
	ticksPerQuarterNote := float64(state.stepSize) / 4.
	if state.triplets {
		ticksPerQuarterNote = float64(state.stepSize) / 4. / 2. * 3
	}
	ticksPerSec := (state.bpm * ticksPerQuarterNote) / 60.0
	tickDuration := uint64(c.sampleRate / ticksPerSec)

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
		if state.step >= state.numSteps() {
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
				voice.pos = sum(m.sum[0:], voice.buf, voice.pos, gain, voice.env.value)
			}
		}

		// trigger a new voice
		if tick && snd.pattern[state.step] != 0 && !snd.muted {
			voice := snd.findFreeVoice()
			voice.buf = snd.buf
			voice.env.startSample = 0
			voice.env.decaySamples = m.clock.sampleRate * snd.decay
			voice.choked = false
			voice.pos = sum(m.sum[offset*2:], voice.buf, 0, gain, voice.env.value)
		}
	}

	if tick {
		state.step++
		if state.step >= state.numSteps() {
			state.step = 0
		}
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
	id   string
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
	buf    []float64
}

func loadSound(path string, numSteps int) (*sound, error) {
	snd := sound{
		file:    path,
		decay:   2,
		gain:    1.,
		pattern: make([]int, numSteps),
		probs:   make([]float64, numSteps),
	}
	for i := range snd.probs {
		snd.probs[i] = 1.
	}
	for i := 0; i < maxVoices; i++ {
		snd.voices = append(snd.voices, &voice{})
	}
	if err := snd.load(path); err != nil {
		return nil, err
	}
	id, err := getSoundID()
	if err != nil {
		return nil, err
	}
	snd.id = id
	return &snd, nil
}

func (s *sound) load(path string) error {
	s.buf = nil

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := wav.NewReader(f)
	for {
		samples, err := r.ReadSamples()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		for _, sample := range samples {
			v := r.FloatValue(sample, 0)
			s.buf = append(s.buf, v) // L
			s.buf = append(s.buf, v) // R
		}
	}

	s.file = path
	return nil
}

func mustLoadSound(path string, numSteps int) *sound {
	snd, err := loadSound(path, numSteps)
	if err != nil {
		panic(err)
	}
	return snd
}

const maxSounds = 64

var soundIDs [maxSounds]struct {
	s     string
	inUse bool
}

func init() {
	const r = 'z' - 'a' + 1
	const a = 'a'
	for i := 0; i < maxSounds; i++ {
		if n := i / r; n > 0 {
			soundIDs[i].s = string(a+n) + string(a+i%r)
		} else {
			soundIDs[i].s = string(a + i)
		}
	}
}

func getSoundID() (string, error) {
	for n, id := range soundIDs {
		if !id.inUse {
			soundIDs[n].inUse = true
			return id.s, nil
		}
	}
	return "", fmt.Errorf("reached maximum number of sounds: %d", maxSounds)
}

func putSoundID(id string) {
	var a, b rune
	a = (rune(id[0]) - 'a')
	if len(id) > 1 {
		b = rune(id[1]) - 'a'*10
	}
	soundIDs[a+b].inUse = false
}
