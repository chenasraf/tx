package picker

import (
	"strings"
	"unicode"
)

// fuzzy returns the rune positions of query in text — a substring if possible, else a
// subsequence — and whether it matched. Smart case: an uppercase letter in query makes it
// case-sensitive.
func fuzzy(query, text string) (map[int]bool, bool) {
	hay, needle := []rune(text), []rune(query)
	if !strings.ContainsFunc(query, unicode.IsUpper) {
		hay, needle = lowerRunes(hay), lowerRunes(needle)
	}
	positions := map[int]bool{}
	if start := indexRunes(hay, needle); start >= 0 {
		for i := start; i < start+len(needle); i++ {
			positions[i] = true
		}
		return positions, true
	}
	j := 0
	for i, r := range hay {
		if j < len(needle) && r == needle[j] {
			positions[i] = true
			j++
		}
	}
	if j < len(needle) {
		return nil, false
	}
	return positions, true
}

// lowering rune by rune keeps positions aligned with the original text
func lowerRunes(rs []rune) []rune {
	out := make([]rune, len(rs))
	for i, r := range rs {
		out[i] = unicode.ToLower(r)
	}
	return out
}

func indexRunes(hay, needle []rune) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// highlight styles text, with matchStyle on the runes at positions.
func highlight(text string, positions map[int]bool, style, matchStyle string) string {
	if len(positions) == 0 {
		return style + text + reset
	}
	var b strings.Builder
	for i, r := range []rune(text) {
		if positions[i] {
			b.WriteString(matchStyle)
		} else {
			b.WriteString(style)
		}
		b.WriteRune(r)
		b.WriteString(reset)
	}
	return b.String()
}
