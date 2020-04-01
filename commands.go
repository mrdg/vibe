package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func clear(s *session, ids []int, args []string) error {
	s.update(func(st *state) {
		for _, id := range ids {
			seq := make([]int, s.state.patternLen)
			st.patterns[id] = seq
		}
	})
	return nil
}

func setp(s *session, ids []int, args []string) error {
	id := ids[0]
	seq := make([]int, s.state.patternLen)
	nodes, err := parsePattern(s.state.timeSig, strings.Join(args, " "))
	if err != nil {
		return err
	}
	s.update(func(st *state) {
		curr := st.patterns[id]
		for _, node := range nodes {
			tmp := make([]int, s.state.patternLen)
			node.sequence(s.state.timeSig, s.state.stepSize, tmp)
			for i, v := range tmp {
				if v != 0 || curr[i] != 0 {
					seq[i] = 1
				}
			}
		}
		st.patterns[id] = seq
	})
	return nil
}

func setn(s *session, ids []int, args []string) error {
	id := ids[0]
	var values []int
	for _, arg := range args {
		v, err := strconv.Atoi(arg)
		if err != nil {
			return err
		}
		if v > s.state.patternLen {
			return fmt.Errorf("out of range: %v", v)
		}
		v--
		values = append(values, v)
	}
	s.update(func(st *state) {
		for _, val := range values {
			st.patterns[id][val] = 1 - st.patterns[id][val]
		}
	})
	return nil
}

func random(s *session, ids []int, args []string) error {
	s.update(func(st *state) {
		for _, id := range ids {
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

func beat(s *session, ids []int, args []string) error {
	timeSig, err := parseTimeSignature(args[0])
	if err != nil {
		return err
	}
	s.update(func(st *state) {
		st.timeSig = timeSig
		st.patternLen = (st.stepSize / timeSig.denom) * timeSig.num
		for i := range st.patterns {
			st.patterns[i] = make([]int, st.patternLen)
		}
	})
	return nil
}

func prob(s *session, ids []int, args []string) error {
	id := ids[0]
	p, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return err
	}
	if p < 0.0 || p > 1.0 {
		return fmt.Errorf("probability is out of range 0-1: %v", p)
	}
	nodes, err := parsePattern(s.state.timeSig, strings.Join(args[1:], " "))
	if err != nil {
		return err
	}
	node := nodes[0]
	tmp := make([]int, s.state.patternLen)
	node.sequence(s.state.timeSig, s.state.stepSize, tmp)
	s.update(func(st *state) {
		for j, v := range tmp {
			if v > 0 {
				st.probs[id][j] = p
			}
		}
	})
	return nil
}

func decay(s *session, ids []int, args []string) error {
	d, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return err
	}
	if d < 0.005 || d > 2 {
		return fmt.Errorf("%v is out of range 5ms - 2s", d)
	}
	s.update(func(st *state) { st.decay[ids[0]] = d })
	return nil
}

func mute(s *session, ids []int, args []string) error {
	s.update(func(st *state) {
		for _, id := range ids {
			st.muted[id] = !st.muted[id]
		}
	})
	return nil
}

func gain(s *session, ids []int, args []string) error {
	db, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return err
	}
	if db > 6 {
		return fmt.Errorf("can't gain by more than 6dB")
	}
	s.update(func(st *state) { st.gain[ids[0]] = db })
	return nil
}

func bpm(s *session, ids []int, args []string) error {
	bpm, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	s.update(func(st *state) { st.bpm = float64(bpm) })
	return nil
}

func exec(s *session, command string) error {
	parts := strings.Split(strings.TrimSpace(command), " ")
	if len(parts) == 0 {
		return fmt.Errorf("invalid command: %v", command)
	}
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}
	name := parts[0]

	for _, cmd := range commands {
		if name != cmd.name {
			continue
		}

		ids, err := parseSoundIDs(s, args, cmd.ids)
		if err != nil {
			return err
		}
		args = args[len(ids):]
		if len(args) < cmd.minArgs {
			return fmt.Errorf("%s: not enough args: %v", cmd.name, args)
		}
		if err := cmd.run(s, ids, args); err != nil {
			return err
		}
	}
	return nil
}

type command struct {
	name    string
	help    string
	run     func(s *session, ids []int, args []string) error
	ids     int // number of ids expected
	minArgs int // min. number of non-id args expected
}

var commands = []command{
	{
		name: "clear",
		run:  clear,
		ids:  -1,
	},
	{
		name:    "setp",
		run:     setp,
		ids:     1,
		minArgs: 1,
	},
	{
		name:    "setn",
		run:     setn,
		ids:     1,
		minArgs: 1,
	},
	{
		name: "rand",
		run:  random,
		ids:  -1,
	},
	{
		name:    "beat",
		run:     beat,
		minArgs: 1,
	},
	{
		name:    "prob",
		run:     prob,
		ids:     1,
		minArgs: 2,
	},
	{
		name:    "bpm",
		run:     bpm,
		minArgs: 1,
	},
	{
		name: "mute",
		run:  mute,
		ids:  -1,
	},
	{
		name:    "gain",
		run:     gain,
		ids:     1,
		minArgs: 1,
	},
	{
		name:    "decay",
		run:     decay,
		ids:     1,
		minArgs: 1,
	},
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

func parseSoundIDs(s *session, args []string, max int) ([]int, error) {
	if max == -1 {
		max = len(args)
	}
	ids := make([]int, max)
	for i := 0; i < max; i++ {
		if len(args[i]) > 1 {
			return nil, fmt.Errorf("not a valid sound id: %s", args[i])
		}
		offset := int(args[i][0])
		A := int('A')
		if offset < A || offset > A+27 {
			return nil, fmt.Errorf("not a valid sound id: %s", args[i])
		}
		id := offset - A
		if id >= len(s.state.samples) {
			return nil, fmt.Errorf("not a valid sound id: %s", args[i])
		}
		ids[i] = id
	}
	return ids, nil
}
