package drop

import "testing"

func TestNewSurfaceIsBlank(t *testing.T) {
	s := NewSurface(3, 2)
	if s.Width() != 3 || s.Height() != 2 {
		t.Fatalf("got %dx%d, want 3x2", s.Width(), s.Height())
	}
	for row := 0; row < 2; row++ {
		for col := 0; col < 3; col++ {
			c := s.At(row, col)
			if c.Content != " " || c.Width != 1 {
				t.Fatalf("(%d,%d) = %#v, want blank", row, col, c)
			}
		}
	}
}

func TestNewSurfaceClampsNegativeDimensions(t *testing.T) {
	s := NewSurface(-1, -5)
	if s.Width() != 0 || s.Height() != 0 {
		t.Fatalf("got %dx%d, want 0x0", s.Width(), s.Height())
	}
}

func TestAtOutOfBoundsReturnsZeroCell(t *testing.T) {
	s := NewSurface(2, 2)
	cases := [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}, {5, 5}}
	for _, c := range cases {
		if got := s.At(c[0], c[1]); got != (Cell{}) {
			t.Errorf("At(%d,%d) = %#v, want zero Cell", c[0], c[1], got)
		}
	}
}

func TestWriteStringASCII(t *testing.T) {
	s := NewSurface(5, 1)
	s.WriteString(0, 0, "abc", Style{})
	want := "abc  " // rest stays blank
	got := ""
	for col := 0; col < 5; col++ {
		got += s.At(0, col).Content
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWriteStringWideCharPlacesContinuation(t *testing.T) {
	s := NewSurface(3, 1)
	s.WriteString(0, 0, "字a", Style{})

	primary := s.At(0, 0)
	if primary.Content != "字" || primary.Width != 2 {
		t.Fatalf("(0,0) = %#v, want primary width-2 字", primary)
	}
	cont := s.At(0, 1)
	if !cont.isContinuation() {
		t.Fatalf("(0,1) = %#v, want continuation placeholder", cont)
	}
	a := s.At(0, 2)
	if a.Content != "a" || a.Width != 1 {
		t.Fatalf("(0,2) = %#v, want 'a' width-1", a)
	}
}

// TestOverwritePrimaryClearsOldContinuation is the write invariant from
// SPEC.md R5 in its most important direction: overwriting a wide cell's
// primary column with a narrow character must also clear the old
// continuation column, not leave it sitting there as an orphaned
// half-glyph with no primary claiming it anymore.
func TestOverwritePrimaryClearsOldContinuation(t *testing.T) {
	s := NewSurface(3, 1)
	s.WriteString(0, 0, "字", Style{}) // primary at 0, continuation at 1

	s.WriteString(0, 0, "x", Style{}) // overwrite the primary with width-1

	got0 := s.At(0, 0)
	if got0.Content != "x" || got0.Width != 1 {
		t.Fatalf("(0,0) = %#v, want 'x' width-1", got0)
	}
	got1 := s.At(0, 1)
	if got1.Content != " " || got1.Width != 1 {
		t.Fatalf("(0,1) = %#v, want cleared to blank, got orphaned continuation", got1)
	}
}

// TestOverwriteContinuationClearsOldPrimary is the write invariant's
// other direction: writing directly into what was a continuation column
// must clear the primary to its left too.
func TestOverwriteContinuationClearsOldPrimary(t *testing.T) {
	s := NewSurface(3, 1)
	s.WriteString(0, 0, "字", Style{}) // primary at 0, continuation at 1

	s.WriteString(0, 1, "y", Style{}) // write directly into the continuation column

	got0 := s.At(0, 0)
	if got0.Content != " " || got0.Width != 1 {
		t.Fatalf("(0,0) = %#v, want cleared to blank, got orphaned primary", got0)
	}
	got1 := s.At(0, 1)
	if got1.Content != "y" || got1.Width != 1 {
		t.Fatalf("(0,1) = %#v, want 'y' width-1", got1)
	}
}

// TestWideCharSkippedAtRightEdge ensures a wide cluster whose second
// column would fall outside the surface is skipped entirely, rather
// than placing a primary cell with no continuation inside the grid.
func TestWideCharSkippedAtRightEdge(t *testing.T) {
	s := NewSurface(3, 1)
	s.WriteString(0, 0, "ab", Style{}) // fills columns 0,1; column 2 is last

	s.WriteString(0, 2, "字", Style{}) // wide char at the last column — no room

	got := s.At(0, 2)
	if got.Content != " " || got.Width != 1 {
		t.Fatalf("(0,2) = %#v, want left blank (wide char skipped, not truncated)", got)
	}
}

func TestWriteStringStopsAtRightEdgeWithoutWrapping(t *testing.T) {
	s := NewSurface(3, 2)
	s.WriteString(0, 0, "abcdef", Style{}) // 6 chars, only 3 columns on row 0

	row0 := ""
	for col := 0; col < 3; col++ {
		row0 += s.At(0, col).Content
	}
	if row0 != "abc" {
		t.Fatalf("row 0 = %q, want %q", row0, "abc")
	}
	// row 1 must be untouched — no wrapping
	row1 := ""
	for col := 0; col < 3; col++ {
		row1 += s.At(1, col).Content
	}
	if row1 != "   " {
		t.Fatalf("row 1 = %q, want unchanged blank %q", row1, "   ")
	}
}

func TestWriteStringOutOfBoundsRowIsNoOp(t *testing.T) {
	s := NewSurface(3, 1)
	s.WriteString(-1, 0, "x", Style{})
	s.WriteString(5, 0, "x", Style{})
	// nothing should have panicked, and row 0 stays blank
	if s.At(0, 0).Content != " " {
		t.Fatalf("row 0 was modified by an out-of-bounds row write")
	}
}

func TestClearResetsSurface(t *testing.T) {
	s := NewSurface(2, 2)
	s.WriteString(0, 0, "字", Style{})
	s.Clear(Style{})
	for row := 0; row < 2; row++ {
		for col := 0; col < 2; col++ {
			c := s.At(row, col)
			if c.Content != " " || c.Width != 1 {
				t.Fatalf("(%d,%d) = %#v after Clear, want blank", row, col, c)
			}
		}
	}
}
