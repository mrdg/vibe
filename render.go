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

	const spacePerStep = 4
	const maxSampleLen = 8

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
		id := "\033[32m" + string(i+int('A')) + "\033[0m"
		fmt.Fprintf(w, "%s %s %s %s\n\n", id, sample, speaker, sb.String())
	}
}

func formatSampleName(sample string, max int) string {
	sample = filepath.Base(sample)
	sample = sample[:len(sample)-len(filepath.Ext(sample))]

	if len(sample) > max {
		sample = sample[:max-1]
		sample += "…"
	}
	if len(sample) < max {
		sample += strings.Repeat(" ", max-len(sample))
	}
	return "\033[34m" + sample + "\033[0m"
}

func numIcon(n int) string {
	if n < 0 || n > 9 {
		panic("number out of range")
	}
	// https://www.unicode.org/emoji/charts/full-emoji-list.html#0030_fe0f_20e3
	return string([]byte{48 + byte(n), 239, 184, 143, 226, 131, 163})
}
