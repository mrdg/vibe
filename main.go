package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gordonklaus/portaudio"
	"github.com/mrdg/ringo/dub"
)

func main() {
	var (
		bpm   = flag.Float64("bpm", 120, "")
		beat  = flag.String("beat", "7/8", "")
		files = flag.String("sounds", "*.wav", "")
		run   = flag.String("run", "", "")
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

	patternLen := (stepSize / timeSig.denom) * timeSig.num

	soundFiles, err := filepath.Glob(*files)
	if err != nil {
		log.Fatal(err)
	}

	var sounds []*sound
	for _, file := range soundFiles {
		sounds = append(sounds, mustLoadSound(file, patternLen))
	}

	var commands []string
	if *run != "" {
		f, err := os.Open(*run)
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			commands = append(commands, strings.TrimSpace(scanner.Text()))
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}

	if err := portaudio.Initialize(); err != nil {
		log.Fatal(err)
	}

	session := &session{
		machine: &machine{
			clock: &clock{sampleRate: sampleRate},
			sum:   make([]float64, bufferSize*nChannels),
			hits:  make([]int, len(soundFiles)),
		},
		state: state{
			bufferSize: bufferSize,
			bpm:        *bpm,
			timeSig:    timeSig,
			sounds:     sounds,
			patternLen: patternLen,
			stepSize:   stepSize,
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

	for _, line := range commands {
		cmd, err := dub.Parse(line)
		if err != nil {
			log.Fatal(err)
		}
		if err := eval(session, cmd); err != nil {
			log.Fatal(err)
		}
	}

	if err := repl(session, os.Stdin); err != nil {
		fmt.Println(err)
		os.Exit(1)
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
	sounds     []*sound
	step       int
	stepSize   int
	patternLen int
}

type savedState struct {
	Sounds []struct {
		Sample  string `json:"sample"`
		Pattern []int  `json:"pattern"`
	}
}

func (s *session) process(out []float32) {
	s.mu.Lock()
	s.machine.process(&s.state, out)
	s.mu.Unlock()
}

func (s *session) update(f func(*state)) {
	s.mu.Lock()
	f(&s.state)
	s.mu.Unlock()
}
