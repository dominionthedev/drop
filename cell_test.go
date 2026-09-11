package drop

import "testing"

func TestCellsForStringASCII(t *testing.T) {
	cells := CellsForString("abc", Style{})
	want := []Cell{
		{Content: "a", Width: 1},
		{Content: "b", Width: 1},
		{Content: "c", Width: 1},
	}
	assertCells(t, cells, want)
}

func TestCellsForStringWideCharGetsContinuation(t *testing.T) {
	// "字" (CJK, width 2) followed by "a" (width 1). The wide cluster
	// must produce a primary cell plus a distinct continuation
	// placeholder, not just a single width-2 cell — the grid stays
	// addressable by column, so every column needs its own Cell.
	cells := CellsForString("字a", Style{})
	want := []Cell{
		{Content: "字", Width: 2},
		{Content: "", Width: 0},
		{Content: "a", Width: 1},
	}
	assertCells(t, cells, want)
}

func TestCellsForStringCombiningMarkStaysOneCell(t *testing.T) {
	// "e" + combining acute accent (U+0301) is two codepoints but one
	// grapheme cluster and must produce exactly one Cell, not two.
	decomposed := "e\u0301"
	precomposed := "\u00e9" // é, single codepoint

	for _, s := range []string{decomposed, precomposed} {
		cells := CellsForString(s, Style{})
		if len(cells) != 1 {
			t.Fatalf("%q: got %d cells, want 1: %#v", s, len(cells), cells)
		}
		if cells[0].Width != 1 {
			t.Fatalf("%q: got width %d, want 1", s, cells[0].Width)
		}
	}
}

func TestCellsForStringFlagEmojiIsOneWideCell(t *testing.T) {
	// A regional-flag emoji is two regional-indicator codepoints that
	// cluster into a single grapheme, width 2.
	usFlag := "\U0001F1FA\U0001F1F8"
	cells := CellsForString(usFlag, Style{})
	want := []Cell{
		{Content: usFlag, Width: 2},
		{Content: "", Width: 0},
	}
	assertCells(t, cells, want)
}

func TestCellsForStringZWJFamilyEmojiIsOneWideCell(t *testing.T) {
	// A ZWJ-joined family emoji is seven codepoints (four people, three
	// zero-width joiners) that cluster into a single grapheme, width 2.
	family := "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"
	cells := CellsForString(family, Style{})
	if len(cells) != 2 {
		t.Fatalf("got %d cells, want 2 (primary + continuation): %#v", len(cells), cells)
	}
	if cells[0].Content != family || cells[0].Width != 2 {
		t.Fatalf("primary cell = %#v, want Content=%q Width=2", cells[0], family)
	}
	if cells[1].Width != 0 || cells[1].Content != "" {
		t.Fatalf("continuation cell = %#v, want Width=0 Content=\"\"", cells[1])
	}
}

func TestCellsForStringEmpty(t *testing.T) {
	if cells := CellsForString("", Style{}); cells != nil {
		t.Fatalf("expected nil for empty string, got %#v", cells)
	}
}

func TestCellsForStringMixed(t *testing.T) {
	cells := CellsForString("a字b", Style{})
	want := []Cell{
		{Content: "a", Width: 1},
		{Content: "字", Width: 2},
		{Content: "", Width: 0},
		{Content: "b", Width: 1},
	}
	assertCells(t, cells, want)
}

func TestCellsForStringLengthMatchesStringWidth(t *testing.T) {
	cases := []string{
		"abc",
		"字a",
		"e\u0301",
		"\U0001F1FA\U0001F1F8",
		"\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466",
		"a字b",
		"",
		"hello, 世界",
	}
	for _, s := range cases {
		got := len(CellsForString(s, Style{}))
		want := StringWidth(s)
		if got != want {
			t.Errorf("%q: len(CellsForString) = %d, StringWidth = %d, want equal", s, got, want)
		}
	}
}

func TestCellIsContinuation(t *testing.T) {
	primary := Cell{Content: "字", Width: 2}
	cont := Cell{Content: "", Width: 0}
	normal := Cell{Content: "a", Width: 1}

	if primary.isContinuation() {
		t.Error("primary cell reported as continuation")
	}
	if !cont.isContinuation() {
		t.Error("continuation cell not reported as continuation")
	}
	if normal.isContinuation() {
		t.Error("normal width-1 cell reported as continuation")
	}
}

func TestBlankCell(t *testing.T) {
	b := blankCell(Style{})
	if b.Content != " " || b.Width != 1 {
		t.Fatalf("blankCell() = %#v, want Content=\" \" Width=1", b)
	}
}

// assertCells compares content/width, ignoring Style since Style{} is
// currently a zero-field placeholder (SPEC.md R5) with nothing to
// meaningfully compare.
func assertCells(t *testing.T, got, want []Cell) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d cells, want %d\ngot:  %#v\nwant: %#v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i].Content != want[i].Content || got[i].Width != want[i].Width {
			t.Fatalf("cell %d: got {%q, %d}, want {%q, %d}",
				i, got[i].Content, got[i].Width, want[i].Content, want[i].Width)
		}
	}
}
