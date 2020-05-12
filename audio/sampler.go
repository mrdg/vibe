package audio

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/youpy/go-wav"
)

const PropSoundMap = "sounds.map"
const numKeys = 25

func Sampler(props *Props) *Instrument {
	sounds := props.MustRegister(PropSoundMap, setSoundMapping, &SoundMapping{})
	var perKeyProps [numKeys]keyProps
	for n := 0; n < numKeys; n++ {
		note := strconv.Itoa(rootPitch + n)
		var kp keyProps
		kp.envAttack = props.MustRegister("env.attack."+note, setEnvParam, 0.0005)
		kp.envDecay = props.MustRegister("env.decay."+note, setEnvParam, 5.0)
		kp.level = props.MustRegister("level."+note, setLevel, 0.)
		kp.choke = props.MustRegister("choke."+note, setInt, 0)
		perKeyProps[n] = kp
	}
	voices := make([]Voice, numVoices)
	for n := range voices {
		voices[n] = &samplerVoice{
			state:    stateFree,
			sounds:   sounds,
			keyProps: perKeyProps,
			env:      &envelope{},
		}
	}
	inst := NewInstrument(props, voices)
	return inst
}

type samplerVoice struct {
	sounds   *atomic.Value
	keyProps [numKeys]keyProps
	state    voiceState
	env      *envelope
	buf      []float64
	pos      int
	pitch    int
}

func (v *samplerVoice) PlayNote(pitch, velocity, duration int) {
	mapping := v.sounds.Load().(*SoundMapping)
	snd := mapping[pitch-rootPitch]
	if snd == nil {
		log.Printf("sampler: no sound mapped to pitch %d", pitch)
		return
	}
	props := v.keyProps[pitch-rootPitch]
	v.buf = snd.buf
	v.state = stateActive
	v.env.attack = props.envAttack.Load().(float64)
	v.env.decay = props.envDecay.Load().(float64)
	v.env.startAttack()
	v.pitch = pitch
}

func (v *samplerVoice) Notify(pitch int) {
	if v.state != stateActive {
		return
	}
	props := v.keyProps[v.pitch-rootPitch]
	if props.choke.Load().(int) == pitch {
		v.stop()
	}
}

func (v *samplerVoice) Process(buf []float64) {
	level := v.keyProps[v.pitch-rootPitch].level.Load().(float64)
	gain := math.Pow(10, level/20.0)

	n := len(buf)
	if nsamples := len(v.buf) - v.pos; nsamples < n {
		n = nsamples
	}
	for i := range buf[:n] {
		buf[i] += v.buf[v.pos] * v.env.value() * gain
		v.pos++
	}
	if v.pos >= len(v.buf) {
		v.buf = nil
		v.pos = 0
		v.state = stateFree
		v.pitch = 0
	}
}

func (v *samplerVoice) stop() {
	v.env.decayRate = 1.0 / (0.001 / sampleRate)
}

func (v *samplerVoice) State() voiceState { return v.state }

// keyProps stores the properties for a single key.
type keyProps struct {
	envAttack *atomic.Value
	envDecay  *atomic.Value
	level     *atomic.Value
	choke     *atomic.Value
}

type Sound struct {
	buf  []float64
	file string
}

const rootPitch = 60

type SoundMapping [numKeys]*Sound

func (m *SoundMapping) Put(key int, snd *Sound) {
	m[key-rootPitch] = snd
}

func setSoundMapping(v interface{}, dest *atomic.Value) error {
	if m, ok := v.(*SoundMapping); ok {
		dest.Store(m)
		return nil
	} else {
		return fmt.Errorf("property value is not a sound mapping: %v", v)
	}
}

func LoadSound(file string) (*Sound, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	snd := Sound{file: file}
	r := wav.NewReader(f)
	for {
		samples, err := r.ReadSamples()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		for _, sample := range samples {
			snd.buf = append(snd.buf, r.FloatValue(sample, 0))
		}
	}
	return &snd, nil
}
