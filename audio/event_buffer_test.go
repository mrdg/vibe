package audio

import (
	"context"
	"testing"
)

func TestEventBufferOffset(t *testing.T) {
	buf := newEventBuffer(8)
	buf.push(event{offset: 2})
	buf.push(event{offset: 3})

	var events []event
	buf.iter(2, func(ev event) {
		events = append(events, ev)
	})
	if want, got := 0, len(events); want != got {
		t.Errorf("expected zero events, got %v", got)
	}

	buf.iter(4, func(ev event) {
		events = append(events, ev)
	})
	if want, got := 2, len(events); want != got {
		t.Errorf("expected %v events, got %v", want, got)
	}
}

func TestEventBuffer(t *testing.T) {
	buf := newEventBuffer(8)

	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	var events []event
	go func() {
		for {
			select {
			case <-ctx.Done():
				buf.iter(-1, func(ev event) {
					events = append(events, ev)
				})
				done <- struct{}{}
				return
			default:
				buf.iter(-1, func(ev event) {
					events = append(events, ev)
				})
			}
		}
	}()

	const numEvents = 1_000_000
	for n := 0; n < numEvents; n++ {
		buf.push(event{offset: n})
	}

	cancel()
	<-done

	if len(events) != numEvents {
		t.Errorf("wrong number of events: want %v, got %v", numEvents, len(events))
	}

	prev := -1
	for _, ev := range events {
		if want, got := prev+1, ev.offset; want != got {
			t.Errorf("discontinuous event offset: want: %v, got %v", want, ev.offset)
		}
		prev++
	}
}
