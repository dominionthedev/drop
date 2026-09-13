package drop

import "sort"

// Screen is the primary object of drop (SPEC.md R1): the composed
// visual state intended for terminal presentation. A Screen holds
// multiple independently constructed Surfaces (R6), each placed at a
// position and painted in z order (R8 — paint order only, not
// geometry), and resolves them into a single flat Surface via Resolve.
//
// A Screen has no terminal I/O dependency (R2): every operation here
// works on a Screen with nothing attached.
type Screen struct {
	width, height int
	layers        []layer
}

// layer is one Surface composed onto a Screen: where it sits, what
// order it paints in, and whether it's currently included at all.
type layer struct {
	name    string
	surface *Surface
	x, y    int
	z       int
	visible bool
}

// NewScreen creates a blank width x height Screen with no layers.
// Negative dimensions are clamped to 0.
func NewScreen(width, height int) *Screen {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return &Screen{width: width, height: height}
}

func (s *Screen) Width() int  { return s.width }
func (s *Screen) Height() int { return s.height }

// AddLayer composes surface into the Screen at (x, y), painted at order
// z — a higher z paints later, i.e. on top of lower-z layers where they
// overlap. name identifies the layer for later lookup with SetVisible or
// RemoveLayer; SPEC.md R6's own example names (background, main,
// sidebar, overlay, notification) are exactly this kind of identifier.
// Adding a layer under a name that already exists replaces it, rather
// than stacking a second layer under the same name.
func (s *Screen) AddLayer(name string, surface *Surface, x, y, z int) {
	l := layer{name: name, surface: surface, x: x, y: y, z: z, visible: true}
	for i := range s.layers {
		if s.layers[i].name == name {
			s.layers[i] = l
			return
		}
	}
	s.layers = append(s.layers, l)
}

// RemoveLayer removes the named layer, if present. A no-op otherwise.
func (s *Screen) RemoveLayer(name string) {
	for i := range s.layers {
		if s.layers[i].name == name {
			s.layers = append(s.layers[:i], s.layers[i+1:]...)
			return
		}
	}
}

// SetVisible toggles a layer's inclusion in Resolve without removing it
// from the Screen — its position, z order, and surface content are
// unchanged, it's just excluded from composition while hidden.
func (s *Screen) SetVisible(name string, visible bool) {
	for i := range s.layers {
		if s.layers[i].name == name {
			s.layers[i].visible = visible
			return
		}
	}
}

// Resolve composes every visible layer, in ascending z order, into a
// single flat Surface the size of the Screen — R7's position, overlap,
// ordering, visibility, and clipping, and R8's paint-order layering,
// resolved into one buffer a renderer can present directly.
//
// A later (higher-z) layer's cells overwrite an earlier layer's wherever
// they overlap. Content placed outside the Screen's bounds — including
// a wide cell whose primary would land on-screen but whose continuation
// would not, or vice versa — is clipped, not wrapped or truncated into
// an inconsistent half-written cell; see blit.
func (s *Screen) Resolve() *Surface {
	out := NewSurface(s.width, s.height)

	ordered := make([]layer, len(s.layers))
	copy(ordered, s.layers)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].z < ordered[j].z })

	for _, l := range ordered {
		if !l.visible || l.surface == nil {
			continue
		}
		blit(out, l.surface, l.x, l.y)
	}
	return out
}

// blit copies every cell of src onto dst at offset (x, y), clipping
// anything that falls outside dst's bounds.
//
// A wide cell's primary and continuation are copied as one atomic unit:
// the continuation is only written if the primary was actually placed.
// Without this, a primary clipped off one edge (e.g. by a negative x)
// would still let its continuation get copied to a valid destination
// column, leaving a continuation cell with no primary to its left —
// which breaks the assumption clearPairAt relies on and would corrupt
// unrelated content the next time something writes over it.
func blit(dst, src *Surface, x, y int) {
	for row := 0; row < src.Height(); row++ {
		for col := 0; col < src.Width(); {
			c := src.At(row, col)
			if c.isContinuation() {
				// Shouldn't occur in a well-formed source surface —
				// a continuation is always preceded by the wide cell
				// that owns it — but skip defensively rather than
				// copy a dangling continuation with nothing before it.
				col++
				continue
			}
			if c.Width == 2 {
				if dst.setCell(y+row, x+col, c) && col+1 < src.Width() {
					dst.setCell(y+row, x+col+1, src.At(row, col+1))
				}
				col += 2
				continue
			}
			dst.setCell(y+row, x+col, c)
			col++
		}
	}
}
