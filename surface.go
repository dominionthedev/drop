package drop

// Surface is a rectangular grid of Cells — the unit a Screen composes
// (SPEC.md R6, R7). A Surface has no terminal I/O dependency (R2) and no
// knowledge of any other surface; composition, z-order, and clipping are
// a Screen-level concern layered on top of this, not implemented here.
// A Surface likewise has no notion of its own position — where it sits
// once composed onto a Screen is a composition-time property, not an
// intrinsic one, so it isn't tracked here either.
type Surface struct {
	width, height int
	cells         []Cell
}

// NewSurface creates a blank width x height Surface, every cell a space
// in the zero Style. Negative dimensions are clamped to 0.
func NewSurface(width, height int) *Surface {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	s := &Surface{width: width, height: height, cells: make([]Cell, width*height)}
	s.Clear(Style{})
	return s
}

func (s *Surface) Width() int  { return s.width }
func (s *Surface) Height() int { return s.height }

// index converts (row, col) to the backing slice's flat index, or -1 if
// out of bounds.
func (s *Surface) index(row, col int) int {
	if row < 0 || row >= s.height || col < 0 || col >= s.width {
		return -1
	}
	return row*s.width + col
}

// At returns the Cell at (row, col). Out-of-bounds returns the zero
// Cell.
func (s *Surface) At(row, col int) Cell {
	i := s.index(row, col)
	if i < 0 {
		return Cell{}
	}
	return s.cells[i]
}

// Clear resets every cell to blank in the given style.
func (s *Surface) Clear(style Style) {
	b := blankCell(style)
	for i := range s.cells {
		s.cells[i] = b
	}
}

// WriteString writes str starting at (row, col), segmenting it via
// CellsForString and placing one resulting Cell per column, advancing
// the pen (SPEC.md R5 — not the terminal's real cursor, see R13) by one
// column per Cell. Continuation placeholders from a wide cluster are
// already separate elements in CellsForString's output, so this needs
// no special-casing for them.
//
// Before every write, clearPairAt enforces the write invariant from R5:
// whatever wide-cell pair currently overlaps the column being written
// is cleared back to blank first, so overwriting one half of an
// existing wide cell never leaves a stray orphaned half on screen.
//
// Writing stops cleanly at the surface's right edge — content that
// doesn't fit is neither wrapped nor truncated mid-cluster. If a wide
// cluster's second column would fall outside the surface, the whole
// cluster is skipped rather than placing a primary cell whose
// continuation doesn't exist in the grid.
func (s *Surface) WriteString(row, col int, str string, style Style) {
	if row < 0 || row >= s.height {
		return
	}
	pen := col
	for _, c := range CellsForString(str, style) {
		if pen < 0 || pen >= s.width {
			return
		}
		if c.Width == 2 && pen+1 >= s.width {
			return
		}
		s.clearPairAt(row, pen)
		s.cells[s.index(row, pen)] = c
		pen++
	}
}

// clearPairAt clears the wide-cell pair overlapping (row, col) back to
// blank, if any — the write invariant from SPEC.md R5. If (row, col) is
// a continuation cell, its primary (one column left) is cleared too,
// and if it's a wide primary, its continuation (one column right) is
// cleared too. A plain width-1 cell has nothing extra to clear; the
// caller's own write immediately after handles it.
func (s *Surface) clearPairAt(row, col int) {
	i := s.index(row, col)
	if i < 0 {
		return
	}
	cell := s.cells[i]

	switch {
	case cell.isContinuation():
		if pi := s.index(row, col-1); pi >= 0 {
			s.cells[pi] = blankCell(s.cells[pi].Style)
		}
		s.cells[i] = blankCell(cell.Style)
	case cell.Width == 2:
		s.cells[i] = blankCell(cell.Style)
		if ci := s.index(row, col+1); ci >= 0 {
			s.cells[ci] = blankCell(s.cells[ci].Style)
		}
	}
}
