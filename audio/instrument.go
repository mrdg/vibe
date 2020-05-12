package audio

import (
	"log"
	"math"
	"sync/atomic"
)

const (
	blockSize  = 16 // this gives about 0.35ms accuracy for sequenced events
	sampleRate = 44100
	bufferSize = 512
)

const numVoices = 12

type voiceState int

const (
	stateFree voiceState = iota
	stateActive
	stateReleased
)

type event struct {
	pitch    int
	offset   int
	velocity int
	duration int
}

type Voice interface {
	PlayNote(pitch, velocity, duration int)
	Process(buf []float64)
	State() voiceState
	Notify(pitch int)
}

type Instrument struct {
	*Props
	voices []Voice
	events *eventBuffer
	buf    []float64
	level  *atomic.Value
}

const propLevel = "level"

func NewInstrument(props *Props, voices []Voice) *Instrument {
	instrument := &Instrument{
		events: newEventBuffer(64),
		buf:    make([]float64, bufferSize),
		Props:  props,
		level:  props.MustRegister(propLevel, setLevel, 0.1),
	}
	for _, v := range voices {
		instrument.voices = append(instrument.voices, v)
	}
	return instrument
}

func (i *Instrument) PlayNote(offset, pitch, velocity, duration int) {
	i.events.push(event{
		pitch:    pitch,
		offset:   offset,
		duration: duration,
	})
}

func (i *Instrument) Process(samples [][]float32) {
	for n := 0; n < len(samples[0]); n += blockSize {
		i.events.iter(n+blockSize, func(ev event) {
			for _, voice := range i.voices {
				voice.Notify(ev.pitch)
			}
			voice := i.findFreeVoice()
			if voice == nil {
				// TODO: implement some kind of voice stealing mechanism
				log.Printf("instrument: no free voice available")
				return
			}
			voice.PlayNote(ev.pitch, ev.velocity, ev.duration)
		})
		for _, voice := range i.voices {
			if voice.State() == stateFree {
				continue
			}
			voice.Process(i.buf[n : n+blockSize])
		}
	}
	db := i.level.Load().(float64)
	gain := math.Pow(10, db/20.0)
	for n := range i.buf {
		sample := float32(gain * i.buf[n])
		samples[0][n] += sample
		samples[1][n] += sample
		i.buf[n] = 0
	}
}

func (i *Instrument) findFreeVoice() Voice {
	for _, voice := range i.voices {
		if voice.State() == stateFree {
			return voice
		}
	}
	return nil
}
