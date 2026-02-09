package fzf

import (
	"testing"
)

func TestFuzzyMatch_ExactPrefix(t *testing.T) {
	matched, positions, score := fuzzyMatch("foo", "foobar")
	if !matched {
		t.Fatal("expected match")
	}
	if len(positions) != 3 {
		t.Fatalf("expected 3 positions, got %d", len(positions))
	}
	for i, p := range positions {
		if p != i {
			t.Errorf("position %d: expected %d, got %d", i, i, p)
		}
	}
	if score <= 0 {
		t.Error("expected positive score")
	}
}

func TestFuzzyMatch_ScatteredChars(t *testing.T) {
	matched, positions, _ := fuzzyMatch("fb", "foobar")
	if !matched {
		t.Fatal("expected match")
	}
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}
	if positions[0] != 0 || positions[1] != 3 {
		t.Errorf("expected positions [0, 3], got %v", positions)
	}
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	matched, _, _ := fuzzyMatch("xyz", "foobar")
	if matched {
		t.Error("expected no match")
	}
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	matched, positions, _ := fuzzyMatch("FOO", "foobar")
	if !matched {
		t.Fatal("expected match")
	}
	if len(positions) != 3 {
		t.Fatalf("expected 3 positions, got %d", len(positions))
	}
}

func TestFuzzyMatch_EmptyQuery(t *testing.T) {
	matched, positions, score := fuzzyMatch("", "anything")
	if !matched {
		t.Fatal("expected match for empty query")
	}
	if len(positions) != 0 {
		t.Error("expected no positions for empty query")
	}
	if score != 0 {
		t.Error("expected zero score for empty query")
	}
}

func TestFuzzyMatch_EmptyCandidate(t *testing.T) {
	matched, _, _ := fuzzyMatch("a", "")
	if matched {
		t.Error("expected no match against empty candidate")
	}
}

func TestFuzzyMatch_ConsecutiveBonus(t *testing.T) {
	_, _, scoreConsec := fuzzyMatch("fo", "foobar")
	_, _, scoreScatter := fuzzyMatch("fb", "foobar")
	if scoreConsec <= scoreScatter {
		t.Errorf("expected consecutive score (%d) > scattered score (%d)", scoreConsec, scoreScatter)
	}
}

func TestFuzzyMatch_WordBoundaryBonus(t *testing.T) {
	_, _, scoreBoundary := fuzzyMatch("fb", "foo-bar")
	_, _, scoreMiddle := fuzzyMatch("fb", "fxxbxx")
	if scoreBoundary <= scoreMiddle {
		t.Errorf("expected boundary score (%d) > middle score (%d)", scoreBoundary, scoreMiddle)
	}
}

func TestMatchItem_NameMatch(t *testing.T) {
	item := Item{Key: "test", Name: "foobar", Aliases: []string{"baz"}}
	result, ok := matchItem("foo", item, 0)
	if !ok {
		t.Fatal("expected match")
	}
	if result.source != matchName {
		t.Error("expected name match source")
	}
	if len(result.namePositions) != 3 {
		t.Errorf("expected 3 name positions, got %d", len(result.namePositions))
	}
}

func TestMatchItem_AliasMatch(t *testing.T) {
	item := Item{Key: "test", Name: "foobar", Aliases: []string{"baz", "qux"}}
	result, ok := matchItem("baz", item, 0)
	if !ok {
		t.Fatal("expected match")
	}
	if result.source != matchAlias {
		t.Error("expected alias match source")
	}
	if len(result.aliasMatches) != 2 {
		t.Fatalf("expected 2 aliasMatches entries, got %d", len(result.aliasMatches))
	}
	if len(result.aliasMatches[0].positions) != 3 {
		t.Errorf("expected 3 positions for first alias, got %d", len(result.aliasMatches[0].positions))
	}
	if result.aliasMatches[1].positions != nil {
		t.Error("expected no match for second alias")
	}
}

func TestMatchItem_NamePreferredOverAlias(t *testing.T) {
	item := Item{Key: "test", Name: "abc", Aliases: []string{"abc"}}
	result, ok := matchItem("abc", item, 0)
	if !ok {
		t.Fatal("expected match")
	}
	if result.source != matchName {
		t.Error("expected name match to be preferred over alias")
	}
}

func TestMatchItem_HighlightsBothNameAndAlias(t *testing.T) {
	item := Item{Key: "test", Name: "abc", Aliases: []string{"axe", "abc"}}
	result, ok := matchItem("a", item, 0)
	if !ok {
		t.Fatal("expected match")
	}
	// Name should have position 0 highlighted
	if len(result.namePositions) != 1 || result.namePositions[0] != 0 {
		t.Errorf("expected name position [0], got %v", result.namePositions)
	}
	// Both aliases contain 'a', both should have highlights
	if len(result.aliasMatches) != 2 {
		t.Fatalf("expected 2 aliasMatches, got %d", len(result.aliasMatches))
	}
	if len(result.aliasMatches[0].positions) != 1 {
		t.Errorf("expected 1 position for first alias, got %d", len(result.aliasMatches[0].positions))
	}
	if len(result.aliasMatches[1].positions) != 1 {
		t.Errorf("expected 1 position for second alias, got %d", len(result.aliasMatches[1].positions))
	}
}

func TestMatchItem_NoMatch(t *testing.T) {
	item := Item{Key: "test", Name: "foobar", Aliases: []string{"baz"}}
	_, ok := matchItem("xyz", item, 0)
	if ok {
		t.Error("expected no match")
	}
}

func TestMatchItem_EmptyQuery(t *testing.T) {
	item := Item{Key: "test", Name: "foobar", Aliases: []string{"baz"}}
	result, ok := matchItem("", item, 0)
	if !ok {
		t.Fatal("expected match for empty query")
	}
	if result.score != 0 {
		t.Error("expected zero score for empty query")
	}
}

func TestFilterAndSort_Basic(t *testing.T) {
	items := []Item{
		{Key: "a", Name: "alpha"},
		{Key: "b", Name: "beta"},
		{Key: "c", Name: "gamma"},
	}

	results := filterAndSort(items, "a")
	if len(results) != 3 { // alpha, beta, and gamma all contain 'a'
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// alpha should rank higher (word-boundary match at start)
	if results[0].item.Key != "a" {
		t.Errorf("expected 'a' first, got %q", results[0].item.Key)
	}
}

func TestFilterAndSort_EmptyQuery(t *testing.T) {
	items := []Item{
		{Key: "a", Name: "alpha"},
		{Key: "b", Name: "beta"},
	}

	results := filterAndSort(items, "")
	if len(results) != 2 {
		t.Fatalf("expected all items for empty query, got %d", len(results))
	}
}

func TestFilterAndSort_NoMatches(t *testing.T) {
	items := []Item{
		{Key: "a", Name: "alpha"},
		{Key: "b", Name: "beta"},
	}

	results := filterAndSort(items, "xyz")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
