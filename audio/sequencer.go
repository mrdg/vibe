package audio

import (
	"fmt"
	"math"
	"sync/atomic"
)

// Pulses per quarter note
const PPQN = 960.

type Clip struct {
	Length     int
	instrument Playable
	notes      []note
}

func NewClip(length float64, p Playable) *Clip {
	return &Clip{
		Length:     int(length * PPQN),
		instrument: p,
	}
}

type Playable interface {
	PlayNote(offset, pitch, velocity, duration int)
}

func (c *Clip) AddNote(position float64, pitch int, length float64) {
	if pitch < 1 || pitch > 127 {
		return
	}
	c.notes = append(c.notes, note{
		pos:    int(position * PPQN),
		pitch:  pitch,
		length: length,
	})
}

type note struct {
	pos      int // position of the note measured in PPQN from the start of a clip
	pitch    int // pitch as a midi note number
	velocity int
	length   float64 // note length in beats
}

type Sequencer struct {
	*Props
	bpm         *atomic.Value
	clips       *atomic.Value
	sampleRate  float64
	totalPulses uint64
}

func NewSequencer(props *Props) *Sequencer {
	clips := make(map[string]*Clip)
	seq := &Sequencer{
		Props:      props,
		sampleRate: sampleRate,
		clips:      props.MustRegister("clips", setClips, clips),
		bpm:        props.MustRegister("bpm", setFloat64(0, 500), 120.0),
	}
	return seq
}

func (s *Sequencer) Tick(numSamples int) {
	bpm := s.bpm.Load().(float64)
	clips := s.clips.Load().(map[string]*Clip)

	// The number of pulses to schedule for each buffer will be fractional,
	// because the PPQN is not a multiple of the buffer size. Truncating it
	// causes the next pulse to be a few samples early, but it's not noticeable.
	numPulses := int(math.Floor(PPQN * (bpm / 60.) / (s.sampleRate / float64(numSamples))))
	samplesPerPulse := s.sampleRate / ((bpm * PPQN) / 60.)

	for _, clip := range clips {
		pos := int(s.totalPulses % uint64(clip.Length)) // current position within the clip
		nextPos := pos + numPulses                      // next position within the clip

		for _, note := range clip.notes {
			duration := int(note.length * s.sampleRate / (bpm / 60.))

			if nextPos > clip.Length {
				// We've reached the end of the clip so also check start of clip for notes to schedule.
				if note.pos >= pos || note.pos < nextPos-clip.Length {
					offset := int(math.Round(float64(clip.Length-pos+note.pos) * samplesPerPulse))
					clip.instrument.PlayNote(offset, note.pitch, note.velocity, duration)
				}
			} else {
				if note.pos >= pos && note.pos < nextPos {
					offset := int(math.Round(float64(note.pos-pos) * samplesPerPulse))
					clip.instrument.PlayNote(offset, note.pitch, note.velocity, duration)
				}
			}
		}
	}
	s.totalPulses += uint64(numPulses)
}

func setClips(v interface{}, dest *atomic.Value) error {
	if c, ok := v.(map[string]*Clip); ok {
		dest.Store(c)
		return nil
	}
	return fmt.Errorf("value is not a map of clips: %v", v)
}
