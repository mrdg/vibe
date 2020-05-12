package audio

type envelopeState int

const (
	stateInit envelopeState = iota
	stateAttack
	stateDecay
	stateSustain
	stateRelease
)

type envelope struct {
	attack  float64
	decay   float64
	sustain float64
	release float64

	attackRate  float64
	decayRate   float64
	releaseRate float64

	val   float64
	state envelopeState
}

func (e *envelope) value() float64 {
	switch e.state {
	case stateInit:
		return 0.
	case stateAttack:
		e.val += e.attackRate
		if e.val >= 1 {
			e.val = 1.0
			if e.decayRate > 0 {
				e.state = stateDecay
			} else {
				e.state = stateSustain
			}
		}
	case stateDecay:
		e.val -= e.decayRate
		if e.val <= e.sustain {
			e.val = e.sustain
			e.state = stateSustain
		}
	case stateSustain:
		if e.sustain == 0 {
			e.state = stateInit
		} else {
			e.val = e.sustain
		}
	case stateRelease:
		e.val -= e.releaseRate
		if e.val <= 0 {
			e.val = 0
			e.state = stateInit
		}
	}
	return e.val
}

func (e *envelope) process(buf []float64) {
	for n := range buf {
		buf[n] *= e.value()
	}
}

func (e *envelope) startAttack() {
	e.val = 0
	e.state = stateAttack
	e.attackRate = 1.0 / (e.attack * sampleRate)
	if e.sustain > 0 {
		e.decayRate = 1.0 - e.sustain/(e.decay*sampleRate)
	} else {
		e.decayRate = 1.0 / (e.decay * sampleRate)
	}
}

func (e *envelope) startRelease() {
	e.state = stateRelease
	e.releaseRate = e.val / (e.release * sampleRate)
}
