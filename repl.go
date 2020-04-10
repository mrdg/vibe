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
	{name: "load", run: load},
	{name: "decay", run: decay, soundArgs: 1},
	{name: "gain", run: gain, soundArgs: 1},
	{name: "clear", run: clear, soundArgs: -1},
	{name: "choke", run: choke, soundArgs: -1},
	{name: "rand", run: random, soundArgs: -1},
	{name: "mute", run: mute, soundArgs: -1},
	{name: "delete", run: delete, soundArgs: -1},
}

type command struct {
	name      string
	run       func(*session, []*sound, []dub.Node) error
	soundArgs int
}

func load(s *session, _ []*sound, args []dub.Node) error {
	path, err := stringArg(args, 0)
	if err != nil {
		return err
	}
	snd, err := loadSound(path, s.state.patternLen)
	if err != nil {
		return err
	}
	s.update(func(st *state) {
		st.sounds = append(st.sounds, snd)
	})
	return nil
}

func delete(s *session, sounds []*sound, args []dub.Node) error {
	s.update(func(st *state) {
		for _, snd1 := range sounds {
			for i, snd2 := range st.sounds {
				if snd1.id == snd2.id {
					putSoundID(snd1.id)
					st.sounds = append(st.sounds[:i], st.sounds[i+1:]...)
				}
			}
		}
	})
	return nil
}

func clear(s *session, sounds []*sound, args []dub.Node) error {
	s.update(func(st *state) {
		for _, snd := range sounds {
			snd.pattern = make([]int, st.patternLen)
		}
	})
	return nil
}

func decay(s *session, sounds []*sound, args []dub.Node) error {
	d, err := floatArg(args, 0)
	if err != nil {
		return err
	}
	if d < 0.005 || d > 2 {
		return fmt.Errorf("%v is out of range 5ms - 2s", d)
	}
	s.update(func(st *state) { sounds[0].decay = d })
	return nil
}

func bpm(s *session, sounds []*sound, args []dub.Node) error {
	bpm, err := intArg(args, 0)
	if err != nil {
		return err
	}
	s.update(func(st *state) { st.bpm = float64(bpm) })
	return nil
}

func mute(s *session, sounds []*sound, args []dub.Node) error {
	s.update(func(st *state) {
		for _, snd := range sounds {
			snd.muted = !snd.muted
		}
	})
	return nil
}

func beat(s *session, sounds []*sound, args []dub.Node) error {
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

		for _, snd := range st.sounds {
			diff := len(snd.pattern) - st.patternLen
			switch {
			case diff > 0:
				snd.pattern = snd.pattern[:st.patternLen]
			case diff < 0:
				tmp := snd.pattern
				snd.pattern = make([]int, st.patternLen)
				for i, v := range tmp {
					snd.pattern[i] = v
				}
			}
			prob := make([]float64, st.patternLen)
			for j := range prob {
				prob[j] = 1.0
			}
			snd.probs = prob
		}
	})
	return nil
}

func random(s *session, sounds []*sound, args []dub.Node) error {
	s.update(func(st *state) {
		for _, snd := range sounds {
			for i := range snd.pattern {
				rand.Seed(time.Now().UnixNano())
				snd.pattern[i] = rand.Intn(2)
			}
		}
	})
	return nil
}

func choke(s *session, sounds []*sound, args []dub.Node) error {
	s.update(func(st *state) {
		for _, snd := range sounds {
			snd.chokeGroup = nil
			for _, other := range sounds {
				if snd != other {
					snd.chokeGroup = append(snd.chokeGroup, other)
				}
			}
		}
	})
	return nil
}

func gain(s *session, sounds []*sound, args []dub.Node) error {
	db, err := floatArg(args, 0)
	if err != nil {
		return err
	}
	if db > 6 {
		return fmt.Errorf("can't gain by more than 6dB")
	}
	s.update(func(st *state) { sounds[0].gain = db })
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
		snd, err := getSound(s, cmd.Name)
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
					for i, v := range seq {
						if snd.pattern[i] != 0 || v != 0 {
							snd.pattern[i] = 1
						}
					}
				})
			case dub.Int:
				s.update(func(st *state) {
					step := val
					step--
					if int(step) < len(snd.pattern) {
						snd.pattern[step] = 1 - snd.pattern[step]
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

func getSound(s *session, identifier dub.Identifier) (*sound, error) {
	for _, snd := range s.state.sounds {
		if snd.id == string(identifier) {
			return snd, nil
		}
	}
	return nil, fmt.Errorf("unknown sound: %s", identifier)
}

func resolveSounds(s *session, args []dub.Node, count int) ([]*sound, error) {
	if count == -1 {
		count = len(args)
	}
	var sounds []*sound
	for i, arg := range args {
		if i >= count {
			break
		}
		identifier, ok := arg.(dub.Identifier)
		if !ok {
			return nil, fmt.Errorf("expected identifier")
		}
		snd, err := getSound(s, identifier)
		if err != nil {
			return nil, err
		}
		sounds = append(sounds, snd)
	}
	return sounds, nil
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

func stringArg(args []dub.Node, pos int) (string, error) {
	if pos >= len(args) {
		return "", fmt.Errorf("wrong number of arguments")
	}
	if v, ok := args[pos].(dub.String); ok {
		return string(v), nil
	}
	return "", fmt.Errorf("wrong type for argument %d: expected string", pos)
}
