package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

func renderState(state state, w io.Writer) {
	sig := state.timeSig
	var icons []string
	for i := 1; i <= sig.num; i++ {
		icons = append(icons, numIcon(i))
	}

	var maxNameLen int
	for _, snd := range state.sounds {
		name := displayName(snd.file)
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}
	maxNameLen += 1

	const spacePerStep = 4
	spacing := (state.stepSize/sig.denom)*spacePerStep - 1
	beats := strings.Join(icons, strings.Repeat(" ", spacing))
	fmt.Fprintf(w, strings.Repeat(" ", maxNameLen)+"   ♩  %s\n", beats)

	for i, snd := range state.sounds {
		speaker := "🔈"
		if snd.muted {
			speaker = "🔇"
		}

		var steps string
		for _, v := range snd.pattern {
			step := "⬜️"
			if v > 0 {
				step = "⬛️"
			}
			steps += step + "  "
		}

		sample := formatSampleName(snd.file, maxNameLen)
		id := colorize(string(i+int('a')), colorGreen)
		row := fmt.Sprintf("%s %s %s %s\n", id, sample, speaker, steps)
		if i < len(state.sounds)-1 {
			row += "\n"
		}
		fmt.Fprintf(w, row)
	}

	var numbers string
	for step := 1; step <= state.patternLen; step++ {
		space := spacePerStep - 2
		if step < 9 {
			space++
		}
		numbers += strconv.Itoa(step) + strings.Repeat(" ", space)
	}
	numbers = colorize(numbers, colorMagenta)
	numbers = strings.Repeat(" ", maxNameLen) + "       " + numbers + "\n"
	fmt.Fprintf(w, numbers)
}

func formatSampleName(sample string, max int) string {
	sample = displayName(sample)

	if len(sample) > max {
		sample = sample[:max-1]
		sample += "…"
	}
	if len(sample) < max {
		sample += strings.Repeat(" ", max-len(sample))
	}
	return colorize(sample, colorBlue)
}

func displayName(filename string) string {
	filename = filepath.Base(filename)
	return filename[:len(filename)-len(filepath.Ext(filename))]
}

func numIcon(n int) string {
	// https://www.unicode.org/emoji/charts/full-emoji-list.html#0030_fe0f_20e3
	return string([]byte{48 + byte(n%10), 239, 184, 143, 226, 131, 163})
}

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
)

func colorize(text string, color int) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", color, text)
}
