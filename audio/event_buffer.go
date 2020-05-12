package audio

import (
	"runtime"
	"sync/atomic"
)

// eventBuffer is a lock-free spsc queue.
type eventBuffer struct {
	events      []event
	read, write *uint32
}

func newEventBuffer(size int) *eventBuffer {
	if size <= 0 || size&(size-1) != 0 {
		panic("event buffer size must be a power of 2")
	}
	return &eventBuffer{
		events: make([]event, size),
		read:   new(uint32),
		write:  new(uint32),
	}
}

func (b *eventBuffer) push(ev event) {
	for atomic.LoadUint32(b.write)-atomic.LoadUint32(b.read) == uint32(len(b.events)) {
		runtime.Gosched()
	}
	write := atomic.LoadUint32(b.write)
	b.events[write%uint32(len(b.events))] = ev
	atomic.StoreUint32(b.write, write+1)
}

func (b *eventBuffer) iter(untilOffset int, f func(event)) {
	read := atomic.LoadUint32(b.read)
	write := atomic.LoadUint32(b.write)
	if read == write {
		return
	}
	for read != write {
		event := b.events[read%uint32(len(b.events))]
		if event.offset >= untilOffset && untilOffset != -1 {
			break
		}
		f(event)
		read++
	}
	atomic.StoreUint32(b.read, read)
}
