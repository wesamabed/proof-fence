# Contributing

Contributions should preserve the benchmark's defensive scope and its evidence
hierarchy.

## What a case must do

1. state one security invariant in `case.json`;
2. use synthetic identities and fixtures;
3. include a starter implementation that violates the invariant;
4. include a grader that catches the violation;
5. include a reference implementation for mechanical self-test;
6. include committed mutants under `mutants/` that the grader must kill;
7. avoid private source code or exploit-ready details tied to real
   infrastructure;
8. document what the case does and does not prove, in `limitations`;
9. declare `"decision_model"` and `"exposure": "PUBLIC_PILOT_ONLY"`.

## What a task statement must and must not say

**Must state:** the API and data semantics; the operational goal; what each field
means and which component populates it; the relevant trust properties as facts
about origin and attribution; what the caller does with the result.

**Must not state:** the boolean conjunction; the expected branch; what to return
when a particular field is missing; which evidence source is trustworthy phrased
as a verdict rather than as an origin; the reference solution in prose.

The ambiguity belongs in the evidence, not in the instructions. A vague task is
not the goal — a case must remain well-specified, with one deterministic correct
answer per input. A candidate that transcribes one sentence of the task into Go
must fail.

Name identifiers after **origins**, not trust levels. `SupervisorPassed` beside
`SelfReportPassed` is fine. `TrustedObservedPass` beside `ClaimedPass` is not:
it hands over the answer.

See `docs/CONSTRUCT_VALIDITY.md` for the ten-point review every new or repaired
case is checked against before commit.

## Authority-decision cases

A case whose output is an authority transition uses the four-valued model in
`docs/AUTHORITY_DECISION_MODEL.md`, with identical names, string values, and doc
comments. Cases are isolated modules, so the enum is duplicated rather than
imported; `TestAuthorityCasesShareTheDecisionVocabulary` fails if the copies
drift.

Such a case must commit `always-grant`, `always-retain`, `always-revoke`, and
`always-quarantine` mutants, and its grader must kill all four. Do not force a
parsing or fact-extraction case into four values where that would distort it.

## Before opening a pull request

```bash
gofmt -l internal cmd
go vet ./...
go test ./...
go run ./cmd/proof-fence selftest
go run ./cmd/proof-fence mutation
```

`selftest` must show every starter failing and every reference passing;
`mutation` must report zero survivors.

Case sources carried over unchanged from v0.1 are not gofmt-normalized, so scope
the formatting check to `internal` and `cmd` plus whatever you touched.

## Boundaries

Case IDs are monotonic (`PF-014`, `PF-015`, …). Do not repurpose existing IDs.

Do not add a held-out confirmatory case to this repository. Everything here is
public pilot material and is contaminated for confirmatory measurement by that
fact alone.
