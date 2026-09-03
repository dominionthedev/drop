# drop

## Definition

> **drop is a terminal screen toolkit for Go that provides abstractions for constructing, composing, manipulating, and rendering a virtual screen onto a terminal.**

drop does not model the terminal itself.

The terminal is provided by `leak`.

drop models what is **presented on the terminal**.

```text
             Application
                  │
                  ▼
                 drop
                  │
               Screen
                  │
        composition / rendering
                  │
                  ▼
             leak.Terminal
                  │
                  ▼
          Terminal emulator
```

`leak` answers:

> How do I communicate with and control this terminal?

drop answers:

> **What should this terminal's screen be and how should it be handled?**

That is the fundamental division.

---

## Purpose

drop exists to change the fundamental programming abstraction of terminal interfaces.

A conventional terminal application treats the terminal primarily as a text I/O endpoint:

```text
stdin ──────► application
application ──────► stdout
```

A terminal UI needs something fundamentally different:

```text
                    Screen
                      │
       ┌──────────────┼──────────────┐
       │              │              │
    content        spatiality     presentation
       │              │              │
       └──────────────┼──────────────┘
                      │
                   renderer
                      │
                  Terminal
```

drop's purpose is therefore:

> **To make the terminal usable as a screen rather than merely as a text interface.**

This means an application should be able to construct a visual state, manipulate that state, compose multiple visual regions, and have drop resolve that state into what the terminal can actually display.

The terminal (the model provided by leak) becomes the **rendering target**.

The Screen becomes the **application's visual medium**.

---

## The fundamental idea

> **A terminal is the device. A Screen is the interface.**

```text
leak
    Terminal
       │
       │ renders to / receives from
       ▼
drop
    Screen
       │
       ├── surfaces
       ├── grids
       ├── cells
       ├── layout
       └── cursor
```

Though even this diagram should not be interpreted as the final type hierarchy.

The important thing is the conceptual separation.

---

## Scope

drop is concerned with the **screen domain**.

It should provide primitives for:

- representing screen space
- representing cells
- representing visual surfaces
- composing surfaces
- arranging surfaces
- addressing regions
- managing virtual dimensions
- representing depth/layers
- resolving virtual screen state into terminal-visible space
- rendering screen state
- efficiently updating the terminal from screen state
- representing the screen's cursor relationship
- resolving terminal input coordinates against composed screen state

drop is **not** responsible for:

- terminal protocols
- escape-sequence definitions
- terminal capability detection
- terminal modes
- terminal event parsing
- TTY management
- raw/cbreak configuration
- terminal lifecycle
- terminal input protocol implementation
- click/drag/hover semantics (double-click timing, drag thresholds, hover enter/leave)
- widget identity (Button, Table, Modal, Sidebar, and so on)

Those belong to `leak`, to a higher-level widget library, or to the host application.

Also, **Screen** is not comparable to what Alternate Screen means. Screen is drop's word for the terminal view.

---

## Requirements

### R1 — Screen abstraction

drop MUST provide a Screen abstraction representing the visual state intended for terminal presentation.

The Screen is the primary object of drop.

Applications should not need to construct terminal output directly to construct a screen.

---

### R2 — Screen must be independent of terminal I/O

A Screen MUST be representable independently from an attached terminal.

This allows:

```text
Screen
   ↓
inspect
modify
compose
test
render
```

without requiring terminal I/O for every operation.

This is also what makes off-screen construction possible.

---

### R3 — Screen must be renderable onto a Terminal

drop MUST provide a rendering mechanism capable of taking a Screen and presenting it through a `leak.Terminal`.

The dependency direction should therefore be:

```text
drop → leak
```

not:

```text
leak → drop
```

and drop should not duplicate `leak.Terminal`. Where drop needs a leak type directly — as with mouse input, see R15 — importing it is expected and consistent with this direction; nothing in leak may import drop.

---

### R4 — Screen must represent spatial information

A Screen MUST represent a two-dimensional visual space.

At minimum, the system needs to reason about:

```text
x
y
width
height
bounds
position
```

The terminal's physical display is ultimately a 2D cell surface.

---

### R5 — Cell

drop MUST have a cell-level representation. A Cell represents what occupies one grid position in the screen's spatial representation.

**Column, not character.** A grid position is addressed by *column* — a fixed slot in the row — not by codepoint or by user-perceived character. Those three units are not the same thing:

- **codepoint** — one Unicode scalar value (a Go `rune`)
- **grapheme cluster** — what a person perceives as one character. Frequently more than one codepoint: `"é"` can be `e` + a combining acute accent, a flag emoji is two regional-indicator codepoints, a family emoji is four-plus codepoints joined by ZWJ
- **column** — how many terminal cells that cluster actually occupies: 0 (combining marks, variation selectors), 1 (most text), or 2 (CJK, most emoji)

A grapheme cluster does not map 1:1 to a column, but the grid MUST stay addressable by column — that's what the terminal, the cursor, and diffing all operate on.

**Shape:**

```go
type Cell struct {
    Content string // the full grapheme cluster — not a rune
    Width   uint8  // 1 or 2 columns; 0 marks a continuation placeholder
    Style   Style  // color/attributes carried per cell; fields not yet specified
}
```

`Content` MUST be a `string`, not a `rune` — a rune cannot hold a multi-codepoint cluster. Width MUST be computed once, when content is written, and cached on the cell — not recomputed on every render pass; grapheme segmentation on a hot path is not acceptable.

**Wide clusters: primary + continuation.** The grid stays a clean, rectangular `[]Cell` indexed by `(row, col)` — no variable-width rows, no special-casing for consumers. A wide cluster (`Width == 2`) writes its full content to its leftmost column, the *primary* cell. The column to its right becomes a *continuation placeholder*: structurally present in the array so indexing stays sane, `Width == 0`, no independent content, never rendered or diffed on its own.

```text
col:     0        1        2        3
       ┌────────┬────────┬────────┬────────┐
       │  "字"  │ (cont) │  "a"   │  " "   │
       │ W = 2  │ W = 0  │ W = 1  │ W = 1  │
       └────────┴────────┴────────┴────────┘
```

**Write invariant.** Writing into a continuation cell's column directly, or overwriting a primary cell with something narrower, MUST first clear the whole pair back to blank before writing new content — otherwise a stray half-glyph is left on screen, and behavior across terminal emulators is inconsistent once that happens. This MUST be enforced at the write API (e.g. `Surface.Set(row, col, cluster, style)`), not left for callers to remember. `Cell` itself stays a plain value type; nothing mutates it by hand outside that API.

**Segmentation.** drop SHOULD NOT implement its own grapheme clustering or East Asian Width tables. `github.com/rivo/uniseg` (already a transitive dependency via the wider Bubble Tea ecosystem) implements UAX #29 clustering and width lookup correctly, and this is a large, yearly-revised correctness surface not worth re-deriving.

**Known limitation, not a defect to chase:** emoji width disagreement across terminals is real and unsolved industry-wide — some terminals render certain emoji at width 1 depending on variation selector or font, and legacy `wcwidth` tables disagree with Unicode's own emoji width recommendations. drop follows Unicode's recommended tables via uniseg and documents that some terminals will occasionally disagree; this is not something to solve here.

**Naming note.** The position tracked while composing content into a Surface (advance by `Width` per cluster written, not by 1) is a different concept from the terminal's real cursor (R13). It MUST NOT be called "cursor." This document refers to it as the **pen**.

---

### R6 — Screen must support independent surfaces

A Screen MUST be capable of containing multiple independently constructed surfaces.

For example:

```text
Screen
│
├── background
├── main
├── sidebar
├── overlay
└── notification
```

The purpose is not merely convenience.

It allows the visual composition to become an explicit operation.

---

### R7 — Surfaces must be composable

drop MUST provide a way to compose surfaces into a Screen.

Composition needs to eventually define semantics for things such as:

```text
position
bounds
overlap
ordering
visibility
clipping
```

This is one of the places where drop can potentially go considerably beyond a simple cell buffer.

---

### R8 — Layering

drop SHOULD support ordered visual layers.

For example:

```text
z = 2   dialog
z = 1   application
z = 0   background
```

The final terminal display is the result of resolving those layers by paint order — this is purely an ordering concern, not a geometric one.

A logical canvas larger than the physical viewport, with a scroll offset, is a plausible future feature and is explicitly *not* part of this requirement. If it's pursued later, it should be specified on its own terms rather than folded into "layering."

---

### R9 — Layout

drop MUST provide mechanisms for determining the spatial arrangement of surfaces and screen regions.

But layout should be concerned with **space**, not widgets.

drop shouldn't know what a:

```text
Button
Table
Modal
Sidebar
```

is.

It should know about:

```text
regions
constraints
bounds
position
relationships
```

Higher-level libraries can assign meaning to those regions (like termfx).

Whether this lives inside drop or as its own consumed package is an open question, not resolved here — flagged, not decided.

---

### R10 — Screen-to-terminal projection

drop MUST have a defined stage that resolves the Screen's representation into the physical representation supported by the target terminal.

Conceptually:

```text
                Screen
                  │
          virtual/composed state
                  │
                  ▼
              projection
                  │
                  ▼
             terminal grid
                  │
                  ▼
             leak.Terminal
```

---

### R11 — Rendering

drop MUST translate the resolved screen state into terminal presentation.

Rendering should be separate from screen construction.

That means:

```text
Screen
   │
   ├── construct
   ├── inspect
   ├── modify
   └── compose
            │
            ▼
         Renderer
            │
            ▼
      leak.Terminal
```

---

### R12 — Incremental rendering

drop SHOULD be capable of determining the difference between a previously presented screen and a new screen.

Conceptually:

```text
previous screen
       +
 current screen
       ↓
    difference
       ↓
   presentation
```

This is important for performance, but it should be a rendering concern rather than something that contaminates the Screen abstraction.

---

### R13 — Cursor

The Screen model MUST account for the cursor as part of terminal presentation.

The cursor is not merely another Cell.

It identifies a position in relation to the Screen and therefore needs its own semantics.

This is the real, terminal-visible cursor — distinct from the pen described in R5, which is an internal write-position with no on-screen presence of its own.

Exactly what the cursor's full semantics should be needs a dedicated specification later.

---

### R14 — Resize

drop MUST expose a way to update Screen state in response to a terminal resize.

`leak` already emits resize information (`event.ResizeEvent`, delivered promptly even on an idle terminal — see leak's own fix history). drop does not need to depend on that event type to benefit from it: the host reads it from `leak.ReadEvent()` and relays the dimensions into drop, for example:

```text
leak.ReadEvent()
       │
   event.ResizeEvent{Cols, Rows}
       │
       ▼
   Screen.Resize(cols, rows)
```

This keeps the R3 dependency direction intact — drop's resize entry point takes plain dimensions, not a leak type — while still making sure the work already done in leak to get resize delivery right doesn't get stranded at that layer.

---

### R15 — Mouse position resolution

drop MUST provide a way to resolve a physical terminal coordinate — as reported by `leak.MouseEvent` — into a location within composed screen state: which surface it landed in, and the local coordinate within that surface.

This belongs in drop, not in the host or in leak, because resolving a click requires exactly the composition state — z-order, bounds, clipping — that only drop has. leak has no visibility into surfaces and shouldn't gain any; the host would have to duplicate drop's entire composition stack to answer "what did the user click on" if resolution lived anywhere else.

```go
import "github.com/dominionthedev/leak/event"

type Hit struct {
    Surface  string           // which surface it landed in
    Row, Col int              // local coordinate within that surface
    Raw      event.MouseEvent // leak's original event, unmodified
}
```

Resolution MUST be the same function regardless of the event's Action (click, drag/motion, or scroll) — leak's `MouseEvent` already distinguishes these; drop shouldn't need two separate resolution paths for "click" versus "drag."

Two details this MUST get right, both direct consequences of decisions made elsewhere in this document:

- **Wide cells** (R5): a physical coordinate landing on a continuation placeholder MUST resolve back to the primary cell's column, not report a hit on empty space.
- **Coordinate systems**: SGR mouse reporting is 1-based (consistent with cursor-position-report convention, same as leak's `CursorPositionEvent`); a `[]Cell` grid is 0-based. That conversion happens here, in drop's resolution function — not in leak, which stays protocol-truth, and not duplicated in the host.

Click semantics — double-click timing, drag thresholds, hover enter/leave — are explicitly out of scope, same tier as Button/Table in the Scope section above. drop answers "what was hit"; what that means is the host's problem.

---

### R16 — Screen should be inspectable

A Screen MUST be inspectable without rendering it.

This enables:

```text
testing
debugging
composition
serialization
comparison
off-screen rendering
```

and potentially other uses we haven't identified yet.

---

### R17 — Screen operations should be composable

Screen primitives SHOULD be usable without adopting a complete TUI architecture.

drop should therefore be usable underneath:

```text
Bubble Tea
custom TUI frameworks
text editors
terminal dashboards
games
visualization systems
terminal document renderers
```

rather than being itself a framework for those applications.

---

### R18 — No application lifecycle

drop MUST NOT own the application's event loop, application lifecycle, or state-management architecture.

Again:

```text
Application
    ↓
framework
    ↓
drop
    ↓
leak
```

not the other way around.

---

### R19 — Terminal constraints must remain real

drop MUST ultimately respect the physical constraints of terminal rendering.

No matter how sophisticated the virtual Screen becomes:

```text
                  drop
                   ↓
        ┌────────────────────┐
        │ virtual abstraction│
        └─────────┬──────────┘
                  ↓
             projection
                  ↓
        columns × rows × cells
                  ↓
              terminal
```

drop can make the abstraction richer.
It cannot make a terminal emulator magically become a GPU.

But we should not make them be a limit when we can fix it.

---

## What this makes drop

So the project is currently defined in one sentence as:

> **drop is a screen construction and rendering toolkit that makes the terminal a programmable visual surface rather than a text I/O interface.**

And its purpose in one sentence:

> **drop exists to provide a richer screen abstraction from which terminal interfaces can be constructed, composed, and rendered, while leaving terminal communication and protocol semantics to `leak`.**

And its architectural boundary:

```text
┌──────────────────────────────────────────────┐
│                  Application                 │
├──────────────────────────────────────────────┤
│             TUI / application logic          │
├──────────────────────────────────────────────┤
│                     drop                     │
│                                              │
│  Screen                                      │
│   ├── Surfaces                               │
│   ├── Grids                                  │
│   ├── Cells                                  │
│   ├── Layout                                 │
│   ├── Layers / depth                         │
│   ├── Cursor                                 │
│   ├── Resize                                 │
│   ├── Mouse resolution                       │
│   └── Projection + Rendering                 │
├──────────────────────────────────────────────┤
│                     leak                     │
│                                              │
│ Terminal / protocols / modes / events / I/O  │
├──────────────────────────────────────────────┤
│                Terminal emulator             │
└──────────────────────────────────────────────┘
```

The particularly important thing is that **Screen is the subject of drop, not Terminal**.
