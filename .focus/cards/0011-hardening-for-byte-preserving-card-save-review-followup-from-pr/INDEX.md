---
schema_version: 2
id: 11
uuid: 019dfaf4-77fa-7797-8760-13b4e1ed1a23
title: 'Hardening for byte-preserving card save (review followup from PR #14)'
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-06
tags: [bug, cli, hardening]
---

## Summary

Three real findings from the two-track review of PR #14 that
weren't blocking but should be cleaned up. PR #14 (commit
`ff2afd8`, v0.1.4) closed #0008 and #0009 and the fix is
correct for every card in the current tree, but the test
suite has a gap and the YAML emitter has two narrow drift
cases.

## Contract

### 1. Add a regression test that catches the original `7057ea5` bug

The new tests in `internal/board/scope_test.go` all pass
against the *old* `Marshal` code path because they seed disk
via `b.NewCard()`, which writes canonical Marshal output —
so the on-disk bytes already equal what Marshal would emit
and the round-trip is byte-identical even with the buggy
code.

Add a test that:

- Seeds a non-target card to disk with hand-authored
  mixed-style frontmatter (inline `tags: [a, b]`,
  hand-wrapped paragraphs, list items).
- Runs a state mutation via the full `transition()` against a
  *different* card.
- Asserts the non-target's bytes are byte-identical (`crypto/sha256`
  hash compare).
- Asserts the target's diff is exactly one line.

This test must FAIL when `saveCardLocked` calls `card.Marshal`
and PASS when it calls `card.MarshalUpdate`. Verify both
directions in the same commit.

### 2. Fix `yamlScalar` for values containing `\n` / `\r` / `\t`

Repro: a title or description containing a literal LF gets
emitted as `'with\nnewline'` (single-quoted with raw LF).
yaml.v3 folds that LF to a space on read, so round-trip
drift.

Fix: when `needsQuoting` flags a control character, switch to
double-quoting with proper escapes (`"with\\nnewline"`).
~10 lines in `internal/board/card/update.go`.

### 3. Fix `yamlScalar` for pure-numeric strings

Repro: `title: 123` is emitted unquoted because
`needsQuoting("123")` returns false. yaml.v3 happens to
round-trip the int back into the typed `string` field, but
external YAML readers see an integer.

Fix: add a numeric check to `needsQuoting`:

```go
import "regexp"
var numericRE = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
// in needsQuoting:
if numericRE.MatchString(v) { return true }
```

### 4. Add a unit test for `requireUnchanged` error paths

No production code mutates `Tags`, `Contract`, or `DependsOn`
today, so the guard never fires. Without a test, a future
caller could silently break.

Add a unit test that constructs a Card from `Parse`, mutates
`Tags` (and separately `Contract` and `DependsOn`), calls
`MarshalUpdate`, and asserts the documented error message
fires. Confirm the same flow with the field unchanged
succeeds.

## Done when

- [ ] Regression test seeds non-target with mixed-style
      frontmatter and verifies non-target bytes are
      identical after a state mutation on a different card.
      Test fails under old `Marshal`, passes under
      `MarshalUpdate`.
- [ ] `yamlScalar` double-quotes values with `\n`, `\r`, `\t`
      and emits proper escapes; round-trip via `card.Parse`
      preserves bytes.
- [ ] `yamlScalar` quotes pure-numeric strings; round-trip
      preserves the value as a YAML string (not an int).
- [ ] Unit test asserts `requireUnchanged` error message on
      Tags / Contract / DependsOn mutation.
- [ ] `go test ./...` green.

## Out of scope

- BOM preservation (P2-1 from review). No card has a BOM today.
- Inline frontmatter comment preservation on rewritten lines
  (P2-2 from review). Documented behavior in
  `rewriteScalarValue`.
- `inserts` ordering (P2-4 from review). Cosmetic.
- Caching `hasTopLevelKey` (P2-5 from review). Premature
  optimization.
- The `equalDate` interface signature (P2-6 from review).
  Cosmetic.
- Merge keys / YAML duplicates (P2-7 from review). No card
  uses them.

## Provenance

Filed by iris 2026-05-06 from the two-track review of PR
#14. Review: https://github.com/evanstern/focus/pull/14#issuecomment-4384027404
PR #14 merge commit: `ff2afd8`. Released as v0.1.4.
