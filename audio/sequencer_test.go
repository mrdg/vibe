package audio

import (
	"reflect"
	"testing"
)

type testInstrument struct {
	events []event
}

func (i *testInstrument) PlayNote(offset, pitch, velocity, duration int) {
	i.events = append(i.events, event{
		offset:   offset,
		pitch:    pitch,
		duration: duration,
	})
}

func (i *testInstrument) flush() {
	i.events = nil
}

func TestSequencer(t *testing.T) {
	const sampleRate = 44100
	const bpm = 120.0
	const bufferSize = sampleRate // use a large buffer size to make testing easier
	instrument := &testInstrument{}

	seq := NewSequencer(NewProps())
	if err := seq.Set("bpm", bpm); err != nil {
		t.Fatal(err)
	}

	clip := NewClip(4, instrument)
	clip.AddNote(0, 69, 1)    // first beat
	clip.AddNote(1.25, 73, 1) // 2nd 16th note on second beat

	if err := seq.Set("clips", map[string]*Clip{
		"beat": clip,
	}); err != nil {
		t.Fatal(err)
	}

	seq.Tick(bufferSize)

	if want, got := []event{
		{offset: 0, pitch: 69, duration: 22050},
		{offset: 27563, pitch: 73, duration: 22050},
	}, instrument.events; !reflect.DeepEqual(want, got) {
		t.Errorf("wrong events:\nwant: %+v\ngot:  %+v", want, got)
	}

	instrument.flush()
	seq.Tick(bufferSize)

	if want, got := 0, len(instrument.events); want != got {
		t.Errorf("wanted zero events, got: %v", instrument.events)
	}

	instrument.flush()
	seq.Tick(bufferSize)

	if want, got := []event{
		{offset: 0, pitch: 69, duration: 22050},
		{offset: 27563, pitch: 73, duration: 22050},
	}, instrument.events; !reflect.DeepEqual(want, got) {
		t.Errorf("wrong events:\nwant: %+v\ngot:  %+v", want, got)
	}
}
