package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

func main() {
	var (
		bpm   = flag.Float64("bpm", 120, "")
		beat  = flag.String("beat", "7/8", "")
		files = flag.String("samples", "*.wav", "")
	)
	flag.Parse()

	const (
		sampleRate = 44100
		nChannels  = 2
		bufferSize = 256
		stepSize   = 16
	)

	timeSig, err := parseTimeSignature(*beat)
	if err != nil {
		log.Fatal(err)
	}

	samples, err := filepath.Glob(*files)
	if err != nil {
		log.Fatal(err)
	}

	var sounds []*sound
	for _, sample := range samples {
		sounds = append(sounds, mustLoadSound(sample))
	}

	patternLen := (stepSize / timeSig.denom) * timeSig.num

	var patterns [][]int
	for _ = range samples {
		p := make([]int, patternLen)
		patterns = append(patterns, p)
	}

	if err := portaudio.Initialize(); err != nil {
		log.Fatal(err)
	}

	decay := make([]time.Duration, len(samples))
	for i := range decay {
		decay[i] = time.Second * 2
	}

	session := &session{
		machine: &machine{
			clock:  &clock{sampleRate: sampleRate},
			sounds: sounds,
			sum:    make([]float64, bufferSize*nChannels),
		},
		state: state{
			bufferSize: bufferSize,
			bpm:        *bpm,
			timeSig:    timeSig,
			samples:    samples,
			patternLen: patternLen,
			steps:      make([]int, len(samples)),
			stepSize:   stepSize,
			muted:      make([]bool, len(samples)),
			patterns:   patterns,
			gain:       make([]float64, len(samples)),
			decay:      decay,
		},
	}

	stream, err := portaudio.OpenDefaultStream(0, 2, sampleRate, bufferSize, session.process)
	if err != nil {
		log.Fatal(err)
	}
	session.stream = stream

	if err := stream.Start(); err != nil {
		log.Fatal(err)
	}

	prompt := bufio.NewScanner(os.Stdin)
	for {
		renderState(session.state, os.Stdout)
		fmt.Printf("> ")
		if !prompt.Scan() {
			if err := prompt.Err(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
		}
		cmd, err := parseCommand(prompt.Text())
		if err != nil {
			fmt.Println(err)
		}
		if err := cmd.exec(session); err != nil {
			fmt.Println(err)
		}
	}
}

type session struct {
	stream  *portaudio.Stream
	mu      sync.Mutex
	machine *machine
	state   state
}

type state struct {
	bufferSize int
	bpm        float64
	timeSig    timeSig
	samples    []string
	steps      []int
	stepSize   int
	patterns   [][]int // TODO: rename to sequence to avoid ambiguity with setp patterns
	patternLen int
	muted      []bool
	gain       []float64 // gain in dB
	decay      []time.Duration
}

type savedState struct {
	Sounds []struct {
		Sample  string `json:"sample"`
		Pattern []int  `json:"pattern"`
	}
}

func (s *session) process(out []int16) {
	s.mu.Lock()
	s.machine.process(s.state, out)
	s.mu.Unlock()
}

func (s *session) update(f func(*state)) {
	s.mu.Lock()
	f(&s.state)
	s.mu.Unlock()
}
