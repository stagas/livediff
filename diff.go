package main

import "strings"

type diffKind uint8

const (
	equal diffKind = iota
	add
	remove
)

type diffLine struct {
	kind diffKind
	text string
}

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []diffLine
}

func splitLines(text string) []string {
	result := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}
	return result
}

// diffLines computes a line diff. The bounded matrix prevents a single large
// rewrite from exhausting memory.
func diffLines(before, after string) []diffLine {
	a, b := splitLines(before), splitLines(after)
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}

	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix &&
		a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}

	oldMiddle := a[prefix : len(a)-suffix]
	newMiddle := b[prefix : len(b)-suffix]
	output := make([]diffLine, 0, len(a)+len(b))
	for _, text := range a[:prefix] {
		output = append(output, diffLine{kind: equal, text: text})
	}

	if len(oldMiddle)*len(newMiddle) > 4_000_000 {
		for _, text := range oldMiddle {
			output = append(output, diffLine{kind: remove, text: text})
		}
		for _, text := range newMiddle {
			output = append(output, diffLine{kind: add, text: text})
		}
	} else {
		table := make([][]uint32, len(oldMiddle)+1)
		for i := range table {
			table[i] = make([]uint32, len(newMiddle)+1)
		}
		for i := len(oldMiddle) - 1; i >= 0; i-- {
			for j := len(newMiddle) - 1; j >= 0; j-- {
				if oldMiddle[i] == newMiddle[j] {
					table[i][j] = table[i+1][j+1] + 1
				} else if table[i+1][j] > table[i][j+1] {
					table[i][j] = table[i+1][j]
				} else {
					table[i][j] = table[i][j+1]
				}
			}
		}

		for i, j := 0, 0; i < len(oldMiddle) || j < len(newMiddle); {
			switch {
			case i < len(oldMiddle) && j < len(newMiddle) && oldMiddle[i] == newMiddle[j]:
				output = append(output, diffLine{kind: equal, text: oldMiddle[i]})
				i++
				j++
			case j < len(newMiddle) && (i == len(oldMiddle) || table[i][j+1] > table[i+1][j]):
				output = append(output, diffLine{kind: add, text: newMiddle[j]})
				j++
			default:
				output = append(output, diffLine{kind: remove, text: oldMiddle[i]})
				i++
			}
		}
	}

	for _, text := range a[len(a)-suffix:] {
		output = append(output, diffLine{kind: equal, text: text})
	}
	return output
}

func makeHunks(diff []diffLine, context int) []hunk {
	type lineRange struct{ start, end int }
	var ranges []lineRange
	for index, line := range diff {
		if line.kind == equal {
			continue
		}
		start, end := max(0, index-context), min(len(diff), index+context+1)
		if len(ranges) > 0 && start <= ranges[len(ranges)-1].end {
			ranges[len(ranges)-1].end = max(ranges[len(ranges)-1].end, end)
		} else {
			ranges = append(ranges, lineRange{start: start, end: end})
		}
	}

	result := make([]hunk, 0, len(ranges))
	for _, r := range ranges {
		oldStart, newStart := 1, 1
		for _, line := range diff[:r.start] {
			if line.kind != add {
				oldStart++
			}
			if line.kind != remove {
				newStart++
			}
		}

		selected := diff[r.start:r.end]
		h := hunk{oldStart: oldStart, newStart: newStart, lines: selected}
		for _, line := range selected {
			if line.kind != add {
				h.oldCount++
			}
			if line.kind != remove {
				h.newCount++
			}
		}
		result = append(result, h)
	}
	return result
}
