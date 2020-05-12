package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
	"github.com/mrdg/vibe/audio"
	"github.com/mrdg/vibe/dub"
)

type env struct {
	sequencer *audio.Sequencer
	devices   map[string]audio.Device
}

func (e *env) setProp(device, prop string, v interface{}) error {
	instr, ok := e.devices[device]
	if !ok {
		return fmt.Errorf("unknown device: %s", device)
	}
	return instr.Set(prop, v)
}

func (e *env) getProp(device, prop string) (interface{}, error) {
	instr, ok := e.devices[device]
	if !ok {
		return nil, fmt.Errorf("unknown device: %s", device)
	}
	return instr.Get(prop)
}

func (e *env) eval(input string) (dub.Node, error) {
	command, err := dub.Parse(input)
	if err != nil {
		return nil, err
	}
	name := string(command.Name)
	for _, cmd := range commands {
		if name != cmd.name {
			continue
		}
		if cmd.arity < 0 {
			arity := -cmd.arity
			if len(command.Args) < arity {
				return nil, fmt.Errorf("%s: wrong number of arguments: need at least %v, got %v",
					cmd.name, arity, len(command.Args))
			}
		} else if len(command.Args) != cmd.arity {
			return nil, fmt.Errorf("%s: wrong number of arguments: want %v, got %v",
				cmd.name, cmd.arity, len(command.Args))
		}
		result, err := cmd.run(e, command.Args)
		if err != nil {
			return result, fmt.Errorf("%s error: %w", cmd.name, err)
		}
		return result, nil
	}
	return nil, fmt.Errorf("unknown command: %s", name)
}

func repl(env *env) error {
	rl, err := readline.New("> ")
	if err != nil {
		return err
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err == io.EOF {
			return err
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		if result, err := env.eval(line); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(result)
		}
	}
}

type command struct {
	name  string
	run   func(*env, []dub.Node) (dub.Node, error)
	arity int // -n means len(args) must be >= n
}

var commands = []command{
	{"loop", loopCommand, -3},
	{"set", setCommand, 3},
	{"load-sound", loadSoundCommand, 3},
}

func setCommand(env *env, args []dub.Node) (dub.Node, error) {
	var device, prop string
	if err := readArgs(args[:2], &device, &prop); err != nil {
		return nil, err
	}
	switch v := args[2].(type) {
	case dub.Number:
		return nil, env.setProp(device, prop, float64(v))
	case dub.String:
		return nil, env.setProp(device, prop, string(v))
	case dub.Identifier:
		return nil, env.setProp(device, prop, string(v))
	default:
		return nil, fmt.Errorf("unsupported property type: %v", v)
	}
}

func loadSoundCommand(env *env, args []dub.Node) (dub.Node, error) {
	var device, file string
	var key int
	if err := readArgs(args, &device, &file, &key); err != nil {
		return nil, err
	}
	v, err := env.getProp(device, audio.PropSoundMap)
	if err != nil {
		return nil, err
	}
	sound, err := audio.LoadSound(file)
	if err != nil {
		return nil, err
	}
	mapping, ok := v.(*audio.SoundMapping)
	if !ok {
		return nil, fmt.Errorf("cannot convert %v to sound mapping", v)
	}
	copy := *mapping
	copy.Put(key, sound)
	return nil, env.setProp(device, audio.PropSoundMap, &copy)
}

func loopCommand(env *env, args []dub.Node) (dub.Node, error) {
	var patternName, device string
	var length float64
	var pattern []dub.Node
	if err := readArgs(args, &patternName, &device, &length, &pattern); err != nil {
		return nil, err
	}
	dev, ok := env.devices[device]
	if !ok {
		return nil, fmt.Errorf("unknown device: %s", device)
	}
	playable, ok := dev.(audio.Playable)
	if !ok {
		return nil, fmt.Errorf("device is not playable: %s", device)
	}
	clip := audio.NewClip(length, playable)
	if err := evalPattern(pattern, clip, length, new(float64)); err != nil {
		return nil, err
	}
	v, err := env.getProp("seq", "clips")
	if err != nil {
		return nil, err
	}
	old := v.(map[string]*audio.Clip)
	// copy the map so we don't modify it in place.
	new := make(map[string]*audio.Clip, len(old))
	for k, v := range old {
		new[k] = v
	}
	new[patternName] = clip
	return nil, env.setProp("seq", "clips", new)
}

func evalPattern(pattern dub.Array, clip *audio.Clip, divLength float64, pos *float64) error {
	noteLength := divLength / float64(len(pattern))
	for _, item := range pattern {
		switch v := item.(type) {
		case dub.Number:
			clip.AddNote(*pos, int(v), noteLength)
			*pos += noteLength
		case dub.Tuple:
			for _, item := range v {
				if i, ok := item.(dub.Number); ok {
					clip.AddNote(*pos, int(i), noteLength)
				}
			}
			*pos += noteLength
		case dub.Array:
			if err := evalPattern(v, clip, noteLength, pos); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid %q in pattern %v", v, pattern)
		}
	}
	return nil
}

func readArgs(args []dub.Node, slots ...interface{}) error {
	if len(args) != len(slots) {
		return errors.New("not enough arguments")
	}
	for n, arg := range args {
		dest := slots[n]
		switch p := dest.(type) {
		case *string:
			switch s := arg.(type) {
			case dub.String:
				*p = string(s)
			case dub.Identifier:
				*p = string(s)
			default:
				return fmt.Errorf("argument error: expected a string or identifier")
			}
		case *float64:
			n, ok := arg.(dub.Number)
			if !ok {
				return fmt.Errorf("argument error: expected a number")
			}
			*p = float64(n)
		case *int:
			n, ok := arg.(dub.Number)
			if !ok {
				return fmt.Errorf("argument error: expected a number")
			}
			*p = int(n)
		case *[]dub.Node:
			arr, ok := arg.(dub.Array)
			if !ok {
				return fmt.Errorf("argument error: expected an array")
			}
			*p = arr
		default:
			panic("readArgs: unhandled destination type: " + fmt.Sprint(p))
		}
	}
	return nil
}
