package audio

import (
	"fmt"
	"math"
	"sync/atomic"
)

const (
	propCutoff     = "cutoff"
	propEnvAttack  = "env.attack"
	propEnvDecay   = "env.decay"
	propEnvSustain = "env.sustain"
	propEnvRelease = "env.release"
	propOsc1Wave   = "osc1.wave"
	propOsc2Wave   = "osc2.wave"
)

func Synth(props *Props) *Instrument {
	var (
		cutoff     = props.MustRegister(propCutoff, setFloat64(0, 20_000), 1000.0)
		envAttack  = props.MustRegister(propEnvAttack, setEnvParam, 0.01)
		envDecay   = props.MustRegister(propEnvDecay, setEnvParam, 0.5)
		envSustain = props.MustRegister(propEnvSustain, setFloat64(0, 1), 1.0)
		envRelease = props.MustRegister(propEnvRelease, setEnvParam, 0.1)
		osc1Wave   = props.MustRegister(propOsc1Wave, setWaveform, "saw")
		osc2Wave   = props.MustRegister(propOsc2Wave, setWaveform, "square")
	)
	voices := make([]Voice, numVoices)
	for n := range voices {
		voices[n] = &synthVoice{
			cutoff:     cutoff,
			envAttack:  envAttack,
			envDecay:   envDecay,
			envSustain: envSustain,
			envRelease: envRelease,
			osc1Wave:   osc1Wave,
			osc2Wave:   osc2Wave,
			state:      stateFree,
			osc1:       &osc{},
			osc2:       &osc{},
			filter:     &filter{coefficients: make([]float64, numCoefficients)},
			env:        &envelope{},
			buf:        make([]float64, bufferSize),
		}
	}
	return NewInstrument(props, voices)
}

type synthVoice struct {
	buf           []float64
	cutoff        *atomic.Value
	envAttack     *atomic.Value
	envDecay      *atomic.Value
	envSustain    *atomic.Value
	envRelease    *atomic.Value
	osc1Wave      *atomic.Value
	osc2Wave      *atomic.Value
	osc1          *osc
	osc2          *osc
	filter        *filter
	env           *envelope
	state         voiceState
	pitch         int
	duration      int
	samplesPlayed int
}

func (v *synthVoice) PlayNote(pitch, velocity, duration int) {
	freq := midiToFreq(pitch)
	v.pitch = pitch
	v.duration = duration
	v.samplesPlayed = 0
	v.env.attack = v.envAttack.Load().(float64)
	v.env.decay = v.envDecay.Load().(float64)
	v.env.sustain = v.envSustain.Load().(float64)
	v.env.release = v.envRelease.Load().(float64)
	v.env.startAttack()
	v.state = stateActive

	phaseDelta := freq * twoPi / sampleRate
	v.osc1.setWaveform(v.osc1Wave.Load().(string))
	v.osc1.freq = freq
	v.osc1.phaseDelta = phaseDelta
	v.osc2.setWaveform(v.osc2Wave.Load().(string))
	v.osc2.freq = freq
	v.osc2.phaseDelta = phaseDelta
}

func (v *synthVoice) reset() {
	v.pitch = 0
	v.filter.y1 = 0.
	v.filter.y2 = 0.
	v.osc1.freq = 0
	v.osc1.phaseDelta = 0
	v.osc2.freq = 0
	v.osc2.phaseDelta = 0
	v.state = stateFree
}

func (v *synthVoice) Process(buf []float64) {
	v.filter.calculateCoefficients(v.cutoff.Load().(float64))
	tmp := v.buf[0:len(buf)]
	v.osc1.process(tmp)
	v.osc2.process(tmp)
	v.filter.process(tmp)
	v.env.process(tmp)
	v.samplesPlayed += len(buf)
	for n := range tmp {
		buf[n] += 0.1 * tmp[n]
		tmp[n] = 0
	}
	if v.samplesPlayed >= v.duration && v.state != stateReleased {
		v.state = stateReleased
		v.env.startRelease()
	}
	if v.state == stateReleased && v.env.state == stateInit {
		v.reset()
	}
}

func (v *synthVoice) Notify(pitch int) {
	if v.pitch == pitch {
		v.stop()
	}
}

func (v *synthVoice) stop() {
	if v.state == stateActive {
		v.env.release = 0.001
		v.env.startRelease()
	}
}

func (v *synthVoice) State() voiceState { return v.state }

const (
	twoPi           = 2 * math.Pi
	numCoefficients = 5
)

type osc struct {
	wave       string
	phase      float64
	phaseDelta float64
	freq       float64
	fn         func(float64) float64
}

func (o *osc) process(buf []float64) {
	for n := range buf {
		buf[n] += o.fn(o.phase)
		o.phase += o.phaseDelta
		if o.phase >= twoPi {
			o.phase -= twoPi
		}
	}
}

func (o *osc) setWaveform(s string) {
	switch s {
	case "sine":
		o.fn = math.Sin
	case "saw":
		o.fn = func(phase float64) float64 {
			return (2.0 * o.phase / twoPi) - 1.
		}
	case "square":
		o.fn = func(phase float64) float64 {
			if phase <= math.Pi {
				return 1.0
			} else {
				return -1.0
			}
		}
	case "off":
		o.fn = func(_ float64) float64 { return 0 }
	}
}

func setWaveform(v interface{}, dest *atomic.Value) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("value is not a string: %v", v)
	}
	switch s {
	case "sine", "saw", "square", "off":
		dest.Store(s)
		return nil
	default:
		return fmt.Errorf("not a valid waveform type: %v", s)
	}
}

type filter struct {
	coefficients []float64

	// state
	y1, y2 float64 // y[n-1] y[n-2]
}

// Lowpass filter based on https://www.w3.org/2011/audio/audio-eq-cookbook.html
func (f *filter) process(buf []float64) {
	c0 := f.coefficients[0]
	c1 := f.coefficients[1]
	c2 := f.coefficients[2]
	c3 := f.coefficients[3]
	c4 := f.coefficients[4]

	for n := range buf {
		in := buf[n]
		out := c0*in + f.y1
		buf[n] = out
		f.y1 = c1*in - c3*out + f.y2
		f.y2 = c2*in - c4*out
	}
}

func (f *filter) calculateCoefficients(freq float64) {
	omega := 2 * math.Pi * freq / sampleRate
	cos := math.Cos(omega)
	sin := math.Sin(omega)

	const q = 1
	alpha := sin / (2. * q)

	var b0, b1, b2, a0, a1, a2 float64

	b0 = (1 - cos) / 2
	b1 = 1 - cos
	b2 = b0
	a0 = 1 + alpha
	a1 = -2 * cos
	a2 = 1 - alpha

	f.coefficients[0] = b0 / a0
	f.coefficients[1] = b1 / a0
	f.coefficients[2] = b2 / a0
	f.coefficients[3] = a1 / a0
	f.coefficients[4] = a2 / a0
}

func midiToFreq(note int) float64 {
	f := math.Pow(2, float64((note-69))/12.0) * 440
	return f
}
