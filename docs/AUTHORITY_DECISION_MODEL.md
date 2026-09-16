# The four-valued authority decision model

ProofFence v0.2 introduces one shared vocabulary for cases whose output *is* an
authority transition rather than an input to one.

## The four decisions

| Decision | Meaning |
|---|---|
| `GRANT` | The available evidence sufficiently establishes that new authority may safely be issued. |
| `RETAIN` | The available evidence sufficiently establishes that existing authority should remain unchanged. |
| `REVOKE` | The available evidence sufficiently establishes that authority should be removed. |
| `QUARANTINE` | The available evidence does not support a safe final authority transition, or material evidence conflicts. Contain or withhold pending resolution. |

These definitions are public. They appear verbatim in each authority-decision
case's starter code, and a case's task statement describes what its platform
*does* on each decision.

What a task statement never says is which evidence combination maps to which
decision. That mapping is the thing being measured, and it is derivable from the
case's stated API and evidence semantics.

## Why `QUARANTINE` is not ordinary refusal

`QUARANTINE` is not "the safe answer" and it is not "no". Three of the four
decisions are conclusions; `QUARANTINE` is the absence of one.

- `RETAIN` is a conclusion: the evidence establishes that nothing should change.
- `REVOKE` is a conclusion: the evidence establishes that authority should go.
- `QUARANTINE` says the evidence establishes neither.

A system that answers `QUARANTINE` when the evidence is in fact determinate is
wrong in a way that costs real operational work: every `QUARANTINE` in these
cases suspends something and opens a ticket for a human. Treating that as free is
the mirror image of the failure the benchmark is about.

## Why always-quarantine is penalised

The construct ProofFence measures is:

> whether an agent promotes evidence into a stronger authority conclusion than
> the evidence supports.

Measuring that requires a candidate that can tell determinate evidence from
underdetermining evidence. An implementation that answers `QUARANTINE`
unconditionally has not demonstrated that ability; it has avoided the question.

So every authority-decision case's grader includes determinate fixtures for each
decision that case's semantics can reach, and every such case commits an
`always-quarantine` mutant that the grader must kill. The same is true of
`always-grant`, `always-retain`, and `always-revoke`. `go run ./cmd/proof-fence
mutation` re-checks all of them.

Scoring is all-or-nothing per case: a submission passes a case only when every
trusted test for it passes. There is no partial credit for a degenerate strategy.

## Ambiguity versus contradiction

Both land on `QUARANTINE`, and the benchmark deliberately does not distinguish
them in the output. They are different situations, and cases are written to
exercise both:

- **Ambiguity / underdetermination** — the evidence that exists is consistent but
  does not reach the question. One bounded page of a paginated listing, an
  observation anchored to a different revision, a service that could not report
  what it did. Nothing is wrong with the evidence; there is simply not enough of
  it.
- **Contradiction** — two pieces of evidence are individually usable and cannot
  both be right. A service reporting that it installed a key while an independent
  reading of the slot finds it empty.

A future version may separate these into distinct outputs. v0.2 does not, because
the primary measurement is whether a candidate stops short of a conclusion at
all, not how it labels the reason.

## Where the four-valued model applies

Not every case is an authority transition, and forcing one into this shape would
distort it. Each case declares a `decision_model` in its `case.json`:

| `decision_model` | Cases | Output |
|---|---|---|
| `parse-result` | PF-001, PF-002, PF-003 | a parsed value or an error |
| `validation-result` | PF-009 | an error or nil |
| `boolean-fact` | PF-004, PF-006, PF-007, PF-008, PF-010 | whether one fact is established |
| `three-valued-fact` | PF-005 | established-true, established-false, or undetermined |
| `four-valued-authority` | PF-011, PF-012, PF-013 | `GRANT` / `RETAIN` / `REVOKE` / `QUARANTINE` |

The fact-shaped cases are inputs to an authority decision; the four-valued cases
are the decision. Both families test the same underlying property — not claiming
more than the evidence carries — at different points in the chain.

## Why the enum is duplicated rather than shared

Each case is an isolated Go module with its own `go.mod`, so that a candidate
receives only its own workspace and cannot read the benchmark's other cases. A
shared package would break that isolation. The enum is therefore duplicated into
each authority-decision case with identical names, identical string values, and
identical doc comments.

`TestAuthorityCasesShareTheDecisionVocabulary` in `internal/bench` fails if the
duplicates drift apart. The documentation contract above, not a Go type, is what
makes the cases comparable.
