package drop

import "github.com/rivo/uniseg"

// Style carries per-cell color and text attributes. Its fields are
// intentionally left unspecified here — see SPEC.md R5, which defers
// Style's design on purpose. This placeholder exists so Cell has
// something to hold and so the write invariant (R5) has a value to
// carry through a clear-and-rewrite, not because Style's shape is
// settled. Do not build against this expecting real fields yet.
type Style struct{}

// Cell represents what occupies one grid position in a Surface.
//
// A grid position is addressed by column, not by codepoint or by
// user-perceived character — those are three different units:
//
//   - codepoint: one Unicode scalar value (a Go rune)
//   - grapheme cluster: what a person perceives as one character,
//     frequently more than one codepoint (a combining accent, a
//     ZWJ-joined emoji sequence)
//   - column: how many terminal cells that cluster actually occupies
//
// A cluster does not map 1:1 to a column, but the grid stays
// addressable by column regardless — that's what the terminal, the
// cursor, and diffing all operate on. See SPEC.md R5 for the full
// rationale.
type Cell struct {
	Content string // the full grapheme cluster — not a rune
	Width   uint8  // 1 or 2 columns; 0 marks a continuation placeholder
	Style   Style
}

// blankCell is an empty, single-width cell: a space, carrying style so
// a cleared region still paints its background/attributes correctly.
func blankCell(style Style) Cell {
	return Cell{Content: " ", Width: 1, Style: style}
}

// isContinuation reports whether c is a continuation placeholder — the
// second column of a wide cell, carrying no independent content.
func (c Cell) isContinuation() bool {
	return c.Width == 0
}

// clusterWidth returns the terminal column width of a single grapheme
// cluster (0, 1, or 2), per East Asian Width and emoji presentation
// rules. This wraps uniseg rather than reimplementing Unicode's width
// tables locally — see SPEC.md R5 on why that isn't done here, and its
// note on emoji width disagreement across terminals being a known,
// unsolved limitation rather than a bug in this function.
func clusterWidth(cluster string) uint8 {
	w := uniseg.StringWidth(cluster)
	if w < 0 {
		return 0
	}
	if w > 2 {
		// Not expected for a single grapheme cluster under normal
		// segmentation, but clamp rather than let a cell claim more
		// columns than the primary+continuation model supports.
		return 2
	}
	return uint8(w)
}

// CellsForString segments s into grapheme clusters and returns the
// resulting Cell sequence, laid out per SPEC.md R5: a wide cluster (Width
// == 2) writes its full content to one Cell and is followed by a single
// continuation placeholder (Width == 0, empty Content) so the returned
// slice's length always equals the string's total column width, not its
// rune or cluster count.
//
// This is a pure function — it has no knowledge of any Surface or grid
// position. Writing the result into a grid, and enforcing the
// clear-before-overwrite invariant when doing so, is a Surface-level
// concern (R5, R6) and is not implemented here.
func CellsForString(s string, style Style) []Cell {
	if s == "" {
		return nil
	}

	cells := make([]Cell, 0, len(s))
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		w := clusterWidth(cluster)

		switch w {
		case 0:
			// Zero-width cluster (combining mark not normalized into
			// its base, a variation selector that didn't attach, a
			// stray control character). It occupies no column of its
			// own; drop it rather than emit a Cell with nothing
			// coherent to place.
			continue
		case 2:
			cells = append(cells, Cell{Content: cluster, Width: 2, Style: style})
			cells = append(cells, Cell{Content: "", Width: 0, Style: style})
		default: // 1
			cells = append(cells, Cell{Content: cluster, Width: 1, Style: style})
		}
	}
	return cells
}

// StringWidth returns the total column width s would occupy once
// segmented into cells — equivalent to len(CellsForString(s, Style{}))
// but without allocating the cell slice.
func StringWidth(s string) int {
	return uniseg.StringWidth(s)
}
