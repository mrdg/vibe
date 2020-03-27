package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type command struct {
	name string
	args []string
}

func parseCommand(line string) (command, error) {
	var cmd command
	parts := strings.Split(strings.TrimSpace(line), " ")
	if len(parts) == 0 {
		return cmd, fmt.Errorf("invalid command: %v", line)
	}
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}
	return command{
		name: parts[0],
		args: args,
	}, nil
}

func (c *command) exec(s *session) error {
	var i byte
	switch c.name {
	case "start", "stop":
		break
	default:
		// Map from sound id ('A', 'B' ...) to index
		i = c.args[0][0] - 65
	}

	switch c.name {
	case "clear":
		seq := make([]int, s.state.patternLen)
		s.update(func(st *state) { st.patterns[i] = seq })
	case "setp":
		seq := make([]int, s.state.patternLen)
		nodes, err := parsePattern(s.state.timeSig, strings.Join(c.args[1:], " "))
		if err != nil {
			return err
		}
		s.update(func(st *state) {
			curr := st.patterns[i]
			for _, node := range nodes {
				tmp := make([]int, s.state.patternLen)
				node.sequence(s.state.timeSig, s.state.stepSize, tmp)
				for i, v := range tmp {
					if v != 0 || curr[i] != 0 {
						seq[i] = 1
					}
				}
			}
			st.patterns[i] = seq
		})
	case "setn":
		var values []int
		for _, arg := range c.args[1:] {
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
				st.patterns[i][val] = 1 - st.patterns[i][val]
			}
		})
	case "bpm":
		bpm, err := strconv.Atoi(c.args[0])
		if err != nil {
			return err
		}
		s.update(func(st *state) { st.bpm = float64(bpm) })
	case "rand":
		pattern := make([]int, s.state.patternLen)
		for i := range pattern {
			rand.Seed(time.Now().UnixNano())
			pattern[i] = rand.Intn(2)
		}
		s.update(func(st *state) { st.patterns[i] = pattern })
	case "beat":
		timeSig, err := parseTimeSignature(c.args[0])
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
	case "mute", "unmute":
		s.update(func(st *state) { st.muted[i] = !st.muted[i] })
	case "start":
		return s.stream.Start()
	case "stop":
		return s.stream.Stop()
	default:
		return fmt.Errorf("unsupported command: %v", c.name)
	}
	return nil
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
