# Contributing

This is currently a single-maintainer project, and right now it's
specification-only — there is no implementation yet. Read [SPEC.md](./SPEC.md)
first; it's the actual design surface at this stage.

## Right now (pre-implementation)

The useful contribution at this phase is discussion, not code: open an
issue against a specific requirement in SPEC.md if something is
underspecified, ambiguous, or you think should be scoped differently.
Requirements explicitly flagged as open questions in the spec (layout's
package boundary, for one) are exactly the kind of thing worth raising.

## Once implementation begins

```bash
make check
```

will run `go build`, `go vet`, a `gofmt -l` check, and `go test -race` for
every package — this is exactly what CI runs, so green locally means
green in CI. The same conventions apply as elsewhere in this ecosystem:

- **Commit messages** — conventional commits, kept short: `type(scope):
  what changed`, e.g. `feat(cell): add wide-character continuation`. A
  body explaining *why*, if the summary doesn't make it obvious.
- **Scope discipline** — a PR implementing one requirement shouldn't also
  redesign an adjacent one, even if it seems related. Open a separate
  issue/PR for that.
- **Tests** — new behavior needs a test that would fail without it; a bug
  fix needs a test that reproduces the bug.
- **SPEC.md is upstream of the code.** If an implementation detail
  contradicts what's written there, that's either a bug in the code or a
  gap in the spec worth raising explicitly — not something to quietly
  paper over in one PR.
