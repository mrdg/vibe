package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mrdg/vibe/audio"
)

func main() {
	run := flag.String("run", "", "File containing newline-separated commands")
	flag.Parse()

	seq := audio.NewSequencer(audio.NewProps())
	sam1 := audio.Sampler(audio.NewProps())
	syn1 := audio.Synth(audio.NewProps())
	syn2 := audio.Synth(audio.NewProps())

	env := env{
		sequencer: seq,
		devices: map[string]audio.Device{
			"seq":  seq,
			"syn1": syn1,
			"syn2": syn2,
			"sam1": sam1,
		},
	}

	sink, err := audio.NewSink()
	if err != nil {
		log.Fatal(err)
	}

	sink.AddSources(syn1, syn2, sam1)
	sink.AddTicker(seq)

	if len(*run) != 0 {
		if err := loadFile(&env, *run); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err := sink.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer sink.Stop()

	if err := repl(&env); err != nil {
		sink.Stop()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadFile(e *env, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if _, err := e.eval(strings.TrimSpace(sc.Text())); err != nil {
			return err
		}
	}
	return sc.Err()
}
