package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mrdg/ringo/dub"
)

var builtins = []command{
	{name: "beat", run: beat},
	{name: "bpm", run: bpm},
	{name: "decay", run: decay, soundArgs: 1},
	{name: "gain", run: gain, soundArgs: 1},
	{name: "clear", run: clear, soundArgs: -1},
	{name: "choke", run: choke, soundArgs: -1},
	{name: "rand", run: random, soundArgs: -1},
	{name: "mute", run: mute, soundArgs: -1},
}

type command struct {
	name      string
	run       func(*session, []int, []dub.Node) error
	soundArgs int
}

func clear(s *session, sounds []int, args []dub.Node) error {
	s.update(func(st *state) {
		for _, i := range sounds {
			seq := make([]int, s.state.patternLen)
			st.patterns[i] = seq
		}
	})
	return nil
}

func decay(s *session, sounds []int, args []dub.Node) error {
	d, err := floatArg(args, 0)
	if err != nil {
		return err
	}
	if d < 0.005 || d > 2 {
		return fmt.Errorf("%v is out of range 5ms - 2s", d)
	}
	s.update(func(st *state) { st.decay[sounds[0]] = d })
	return nil
}

func bpm(s *session, sounds []int, args []dub.Node) error {
	bpm, err := intArg(args, 0)
	if err != nil {
		return err
	}
	s.update(func(st *state) { st.bpm = float64(bpm) })
	return nil
}

func mute(s *session, sounds []int, args []dub.Node) error {
	s.update(func(st *state) {
		for _, i := range sounds {
			st.muted[i] = !st.muted[i]
		}
	})
	return nil
}

func beat(s *session, sounds []int, args []dub.Node) error {
	num, err := intArg(args, 0)
	if err != nil {
		return err
	}
	denom, err := intArg(args, 1)
	if err != nil {
		return err
	}
	s.update(func(st *state) {
		st.timeSig = timeSig{num: int(num), denom: int(denom)}
		st.patternLen = (st.stepSize / st.timeSig.denom) * st.timeSig.num
		for i := range st.patterns {
			st.patterns[i] = make([]int, st.patternLen)
			prob := make([]float64, st.patternLen)
			for j := range prob {
				prob[j] = 1.0
			}
			st.probs[i] = prob
		}
	})
	return nil
}

func random(s *session, sounds []int, args []dub.Node) error {
	s.update(func(st *state) {
		for _, id := range sounds {
			pattern := make([]int, s.state.patternLen)
			for i := range pattern {
				rand.Seed(time.Now().UnixNano())
				pattern[i] = rand.Intn(2)
			}
			st.patterns[id] = pattern
		}
	})
	return nil
}

func choke(s *session, sounds []int, args []dub.Node) error {
	s.update(func(st *state) {
		st.choke = make([][]int, len(st.samples))
		for _, id := range sounds {
			st.choke[id] = nil
			for _, other := range sounds {
				if id != other {
					st.choke[id] = append(st.choke[id], other)
				}
			}
		}
	})
	return nil
}

func gain(s *session, sounds []int, args []dub.Node) error {
	db, err := floatArg(args, 0)
	if err != nil {
		return err
	}
	if db > 6 {
		return fmt.Errorf("can't gain by more than 6dB")
	}
	s.update(func(st *state) { st.gain[sounds[0]] = db })
	return nil
}

func repl(session *session, input io.Reader) error {
	prompt := bufio.NewScanner(input)
	for {
		renderState(session.state, os.Stdout)
		fmt.Printf("> ")
		if !prompt.Scan() {
			if err := prompt.Err(); err != nil {
				return err
			}
		}
		command, err := dub.Parse(prompt.Text())
		if err != nil {
			fmt.Println(err)
			continue
		}
		if err := eval(session, command); err != nil {
			fmt.Println(err)
			continue
		}
	}
}

func eval(s *session, cmd dub.Command) error {
	if len(cmd.Name) == 1 {
		id, err := soundIndex(s, cmd.Name)
		if err != nil {
			return err
		}
		for _, arg := range cmd.Args {
			switch val := arg.(type) {
			case dub.MatchExpr:
				num := s.state.timeSig.num
				denom := s.state.timeSig.denom
				seq, err := dub.EvalMatchExpr(val, num, denom, s.state.stepSize)
				if err != nil {
					return err
				}
				s.update(func(st *state) {
					current := st.patterns[id]
					for i, v := range seq {
						if current[i] != 0 || v != 0 {
							current[i] = 1
						}
					}
				})
			case dub.Int:
				s.update(func(st *state) {
					step := val
					step--
					if int(step) < len(st.patterns[id]) {
						st.patterns[id][step] = 1 - st.patterns[id][step]
					}
				})
			default:
				return fmt.Errorf("unexpected argument: %v", arg)
			}
		}
		return nil
	} else {
		for _, command := range builtins {
			if command.name != string(cmd.Name) {
				continue
			}
			sounds, err := resolveSounds(s, cmd.Args, command.soundArgs)
			if err != nil {
				return fmt.Errorf("%s: %s", command.name, err)
			}
			if err := command.run(s, sounds, cmd.Args[len(sounds):]); err != nil {
				return fmt.Errorf("%s: %s", command.name, err)
			}
			return nil
		}
		return fmt.Errorf("unknown function: %s", cmd.Name)
	}
}

func soundIndex(s *session, identifier dub.Identifier) (int, error) {
	// TODO: sound identifiers are just assumed to be single letters for now
	ident := strings.ToLower(string(identifier))
	offset := int(ident[0])
	a := int('a')
	id := offset - a
	if offset < a || offset > a+27 || id >= len(s.state.samples) {
		return id, fmt.Errorf("not a valid sound id: %s", identifier)
	}
	return id, nil
}

func resolveSounds(s *session, args []dub.Node, count int) ([]int, error) {
	if count == -1 {
		count = len(args)
	}
	var ids []int
	for i, arg := range args {
		if i >= count {
			break
		}
		identifier, ok := arg.(dub.Identifier)
		if !ok {
			return nil, fmt.Errorf("expected identifier")
		}
		id, err := soundIndex(s, identifier)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseTimeSignature(s string) (timeSig, error) {
	var t timeSig
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return t, fmt.Errorf("not a valid time signature: %s", s)
	}
	num, err := strconv.Atoi(parts[0])
	if err != nil {
		return t, fmt.Errorf("bad numerator %s: %s", parts[0], err)
	}
	denom, err := strconv.Atoi(parts[1])
	if err != nil {
		return t, fmt.Errorf("bad denominator %s: %s", parts[1], err)
	}
	return timeSig{num: num, denom: denom}, nil
}

func intArg(args []dub.Node, pos int) (int, error) {
	if pos >= len(args) {
		return 0, fmt.Errorf("wrong number of arguments")
	}
	if v, ok := args[pos].(dub.Int); ok {
		return int(v), nil
	}
	return 0, fmt.Errorf("wrong type for argument %d: expected integer", pos)
}

func floatArg(args []dub.Node, pos int) (float64, error) {
	if pos >= len(args) {
		return 0, fmt.Errorf("wrong number of arguments")
	}
	switch v := args[pos].(type) {
	case dub.Float:
		return float64(v), nil
	case dub.Int:
		return float64(int(v)), nil
	default:
		return 0, fmt.Errorf("wrong type for argument %d: expected float", pos)
	}
}
