package drop

import "testing"

func rowString(s *Surface, row int) string {
	out := ""
	for col := 0; col < s.Width(); col++ {
		out += s.At(row, col).Content
	}
	return out
}

func TestResolveEmptyScreenIsBlank(t *testing.T) {
	scr := NewScreen(3, 2)
	out := scr.Resolve()
	if out.Width() != 3 || out.Height() != 2 {
		t.Fatalf("got %dx%d, want 3x2", out.Width(), out.Height())
	}
	if rowString(out, 0) != "   " {
		t.Fatalf("row 0 = %q, want blank", rowString(out, 0))
	}
}

func TestResolveSingleLayer(t *testing.T) {
	scr := NewScreen(5, 1)
	surf := NewSurface(3, 1)
	surf.WriteString(0, 0, "abc", Style{})
	scr.AddLayer("main", surf, 1, 0, 0)

	out := scr.Resolve()
	if got := rowString(out, 0); got != " abc " {
		t.Fatalf("got %q, want %q", got, " abc ")
	}
}

func TestResolveHigherZWinsOnOverlap(t *testing.T) {
	scr := NewScreen(3, 1)

	back := NewSurface(3, 1)
	back.WriteString(0, 0, "AAA", Style{})
	scr.AddLayer("background", back, 0, 0, 0)

	front := NewSurface(1, 1)
	front.WriteString(0, 0, "B", Style{})
	scr.AddLayer("overlay", front, 1, 0, 1) // higher z, only covers the middle column

	out := scr.Resolve()
	if got := rowString(out, 0); got != "ABA" {
		t.Fatalf("got %q, want %q (overlay should win the middle column)", got, "ABA")
	}
}

func TestResolveOrderIndependentOfAddOrder(t *testing.T) {
	// z decides paint order, not the order layers were added in.
	scr := NewScreen(1, 1)
	front := NewSurface(1, 1)
	front.WriteString(0, 0, "F", Style{})
	back := NewSurface(1, 1)
	back.WriteString(0, 0, "B", Style{})

	scr.AddLayer("front", front, 0, 0, 5) // added first, but higher z
	scr.AddLayer("back", back, 0, 0, 0)   // added second, but lower z

	if got := rowString(scr.Resolve(), 0); got != "F" {
		t.Fatalf("got %q, want %q — z order must win regardless of add order", got, "F")
	}
}

func TestResolveClipsPastRightAndBottomEdge(t *testing.T) {
	scr := NewScreen(3, 1)
	surf := NewSurface(5, 1)
	surf.WriteString(0, 0, "abcde", Style{})
	scr.AddLayer("main", surf, 0, 0, 0) // surface wider than the screen

	out := scr.Resolve()
	if got := rowString(out, 0); got != "abc" {
		t.Fatalf("got %q, want %q (clipped at the screen's right edge)", got, "abc")
	}
}

func TestResolveClipsNegativeOffset(t *testing.T) {
	scr := NewScreen(3, 1)
	surf := NewSurface(3, 1)
	surf.WriteString(0, 0, "abc", Style{})
	scr.AddLayer("main", surf, -1, 0, 0) // shifted one column off the left edge

	out := scr.Resolve()
	if got := rowString(out, 0); got != "bc " {
		t.Fatalf("got %q, want %q ('a' clipped off the left edge)", got, "bc ")
	}
}

// TestResolveClippedWidePrimaryDoesNotOrphanContinuation is a regression
// test for exactly the bug blit's doc comment describes: a wide cell
// whose primary is clipped off-screen (via a negative x) must not still
// let its continuation get copied to a valid destination column. A
// naive column-by-column copy would do exactly that, leaving a
// continuation cell with no primary to its left.
func TestResolveClippedWidePrimaryDoesNotOrphanContinuation(t *testing.T) {
	scr := NewScreen(3, 1)
	surf := NewSurface(3, 1)
	surf.WriteString(0, 0, "字a", Style{}) // primary at col 0, continuation at col 1, 'a' at col 2
	scr.AddLayer("main", surf, -1, 0, 0)  // shift left by 1: primary clips off, continuation would land at dst col 0

	out := scr.Resolve()
	got := out.At(0, 0)
	if got.isContinuation() {
		t.Fatalf("dst (0,0) is an orphaned continuation cell: %#v", got)
	}
	if got.Content != " " || got.Width != 1 {
		t.Fatalf("dst (0,0) = %#v, want left blank (clipped, not orphaned)", got)
	}
}

func TestSetVisibleExcludesLayerWithoutRemoving(t *testing.T) {
	scr := NewScreen(3, 1)
	surf := NewSurface(3, 1)
	surf.WriteString(0, 0, "abc", Style{})
	scr.AddLayer("main", surf, 0, 0, 0)

	scr.SetVisible("main", false)
	if got := rowString(scr.Resolve(), 0); got != "   " {
		t.Fatalf("hidden layer got %q, want blank", got)
	}

	scr.SetVisible("main", true)
	if got := rowString(scr.Resolve(), 0); got != "abc" {
		t.Fatalf("re-shown layer got %q, want %q", got, "abc")
	}
}

func TestRemoveLayer(t *testing.T) {
	scr := NewScreen(3, 1)
	surf := NewSurface(3, 1)
	surf.WriteString(0, 0, "abc", Style{})
	scr.AddLayer("main", surf, 0, 0, 0)
	scr.RemoveLayer("main")

	if got := rowString(scr.Resolve(), 0); got != "   " {
		t.Fatalf("got %q after removal, want blank", got)
	}
	// removing again, or a name that never existed, must not panic
	scr.RemoveLayer("main")
	scr.RemoveLayer("never-existed")
}

func TestAddLayerReplacesExistingName(t *testing.T) {
	scr := NewScreen(3, 1)
	first := NewSurface(3, 1)
	first.WriteString(0, 0, "AAA", Style{})
	scr.AddLayer("main", first, 0, 0, 0)

	second := NewSurface(3, 1)
	second.WriteString(0, 0, "BBB", Style{})
	scr.AddLayer("main", second, 0, 0, 0)

	if got := rowString(scr.Resolve(), 0); got != "BBB" {
		t.Fatalf("got %q, want %q — AddLayer under an existing name should replace it", got, "BBB")
	}
	if n := len(scr.layers); n != 1 {
		t.Fatalf("got %d layers, want 1 (replace, not stack)", n)
	}
}
