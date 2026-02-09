package fzf

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// matchSource indicates whether a match was found in the Name or an Alias.
type matchSource int

const (
	matchName matchSource = iota
	matchAlias
)

// aliasMatch holds match positions for a single alias.
type aliasMatch struct {
	positions []int
}

// matchResult holds the result of matching a query against an Item.
type matchResult struct {
	item          Item
	index         int // original index in items slice
	score         int
	namePositions []int        // matched char positions in Name
	aliasMatches  []aliasMatch // per-alias match positions (nil entry = no match)
	source        matchSource
}

// filteredItem is a matchResult ready for display.
type filteredItem = matchResult

// fuzzyMatch performs a case-insensitive, left-to-right fuzzy match of query
// against candidate. It returns whether the query matched, the positions of
// matched characters in the candidate, and a score.
func fuzzyMatch(query, candidate string) (bool, []int, int) {
	if query == "" {
		return true, nil, 0
	}

	lowerCandidate := strings.ToLower(candidate)
	lowerQuery := strings.ToLower(query)

	positions := make([]int, 0, utf8.RuneCountInString(query))
	score := 0
	ci := 0 // candidate rune index
	prevMatchIdx := -1

	candidateRunes := []rune(lowerCandidate)
	queryRunes := []rune(lowerQuery)

	qi := 0
	for ci < len(candidateRunes) && qi < len(queryRunes) {
		if candidateRunes[ci] == queryRunes[qi] {
			positions = append(positions, ci)

			// Consecutive match bonus
			if prevMatchIdx == ci-1 {
				score += 4
			}

			// Word-boundary bonus: first char, or preceded by separator
			if ci == 0 || isSeparator(candidateRunes[ci-1]) {
				score += 3
			}

			// Exact case match bonus
			origRunes := []rune(candidate)
			qOrigRunes := []rune(query)
			if origRunes[ci] == qOrigRunes[qi] {
				score += 1
			}

			prevMatchIdx = ci
			qi++
		}
		ci++
	}

	if qi < len(queryRunes) {
		return false, nil, 0
	}

	// Base score for matching
	score += 1

	return true, positions, score
}

func isSeparator(r rune) bool {
	return r == ' ' || r == '-' || r == '_' || r == '/' || r == '.'
}

// matchItem tries to fuzzy-match query against item's Name and all Aliases.
// It returns a matchResult with highlight positions for every field that matched.
// The score is taken from the best-matching field.
func matchItem(query string, item Item, index int) (matchResult, bool) {
	if query == "" {
		return matchResult{
			item:  item,
			index: index,
			score: 0,
		}, true
	}

	anyMatched := false
	bestScore := 0
	bestSource := matchName

	// Try name
	var namePositions []int
	nameMatched, nPos, nScore := fuzzyMatch(query, item.Name)
	if nameMatched {
		anyMatched = true
		namePositions = nPos
		bestScore = nScore + 10 // bonus for name match
	}

	// Try each alias — always, so we can highlight all that match
	aliasMatches := make([]aliasMatch, len(item.Aliases))
	for i, alias := range item.Aliases {
		matched, positions, score := fuzzyMatch(query, alias)
		if matched {
			aliasMatches[i] = aliasMatch{positions: positions}
			if !anyMatched || score > bestScore {
				bestScore = score
				bestSource = matchAlias
			}
			anyMatched = true
		}
	}

	if !anyMatched {
		return matchResult{}, false
	}

	return matchResult{
		item:          item,
		index:         index,
		score:         bestScore,
		namePositions: namePositions,
		aliasMatches:  aliasMatches,
		source:        bestSource,
	}, true
}

// filterAndSort filters items by query and returns sorted results.
// With an empty query, items are sorted alphabetically by name (A first, i.e.
// lowest index = A). With a non-empty query, items are sorted by match score
// descending (best match at lowest index).
func filterAndSort(items []Item, query string) []filteredItem {
	var results []filteredItem
	for i, item := range items {
		if result, ok := matchItem(query, item, i); ok {
			results = append(results, result)
		}
	}

	if query == "" {
		// Alphabetical by name (case-insensitive)
		sort.SliceStable(results, func(i, j int) bool {
			return strings.ToLower(results[i].item.Name) < strings.ToLower(results[j].item.Name)
		})
	} else {
		// Best match first
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].score > results[j].score
		})
	}

	return results
}
