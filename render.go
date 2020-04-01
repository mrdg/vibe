package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func renderState(state state, w io.Writer) {
	sig := state.timeSig
	var icons []string
	for i := 1; i <= sig.num; i++ {
		icons = append(icons, numIcon(i))
	}

	var maxSampleLen int
	for _, sample := range state.samples {
		sample = displayName(sample)
		if len(sample) > maxSampleLen {
			maxSampleLen = len(sample)
		}
	}
	maxSampleLen += 1

	const spacePerStep = 4
	spacing := (state.stepSize/sig.denom)*spacePerStep - 1
	numbers := strings.Join(icons, strings.Repeat(" ", spacing))
	fmt.Fprintf(w, strings.Repeat(" ", maxSampleLen)+"   ♩  %s\n", numbers)
	for i, pattern := range state.patterns {
		speaker := "🔈"
		if state.muted[i] {
			speaker = "🔇"
		}

		var sb strings.Builder
		for _, v := range pattern {
			step := "⬜️"
			if v > 0 {
				step = "⬛️"
			}
			sb.WriteString(step + "  ")
		}

		sample := formatSampleName(state.samples[i], maxSampleLen)
		id := colorize(string(i+int('A')), colorGreen)
		fmt.Fprintf(w, "%s %s %s %s\n\n", id, sample, speaker, sb.String())
	}
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
	if n < 0 || n > 9 {
		panic("number out of range")
	}
	// https://www.unicode.org/emoji/charts/full-emoji-list.html#0030_fe0f_20e3
	return string([]byte{48 + byte(n), 239, 184, 143, 226, 131, 163})
}

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
)

func colorize(text string, color int) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", color, text)
}
