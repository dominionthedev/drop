# drop

[![CI](https://github.com/dominionthedev/drop/actions/workflows/ci.yml/badge.svg)](https://github.com/dominionthedev/drop/actions/workflows/ci.yml)

**drop** is a terminal screen toolkit for Go — abstractions for
constructing, composing, manipulating, and rendering a virtual screen onto
a terminal.

drop does not talk to the terminal directly. That's [`leak`](https://github.com/dominionthedev/leak)'s job — raw mode, escape sequences, event parsing, I/O. drop models what gets *presented*: a Screen made of Surfaces, Grids, and Cells, composed and projected onto whatever `leak.Terminal` actually supports.

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

## Why not just print styled strings?

Most terminal styling tools (lipgloss and friends) work by producing
strings with escape sequences baked in. That's fine for text output, but
it isn't a screen model — there's no cell grid to diff against, no
z-ordered composition, no addressable position to resolve a click
against. drop is closer in spirit to what `tcell`/`notcurses` do
architecturally — a real cell buffer with incremental rendering — except
as a library that composes underneath an existing event loop (Bubble Tea
or otherwise) instead of owning one itself.

## What drop is not

- Not a terminal protocol / escape-sequence library (that's `leak`)
- Not a widget library — no opinion on what a Button or Table is
- Not a TUI framework — it doesn't own your event loop or app lifecycle
- Not a styling library — Style is a per-cell attribute, not the point

See [SPEC.md](./SPEC.md) for the full requirements (R1–R19): Cell
representation and wide-character handling, surface composition and
layering, layout, screen-to-terminal projection, incremental rendering,
cursor, resize, and mouse position resolution.

## Status

**Specification-only.** SPEC.md is written; no Go implementation exists
yet. `go get` will get you an empty module. Track progress in
[CHANGELOG.md](./CHANGELOG.md).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) — right now that mostly means
raising issues against SPEC.md, not sending code.

## License

MIT - See [LICENSE](./LICENSE)
