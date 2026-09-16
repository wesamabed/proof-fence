# Benchmark card — ProofFence v0.2

## Intended use
Defensive evaluation of coding agents and review workflows on small
security-semantic programming tasks.

## Unit of evaluation
A case is a small isolated Go module with a task statement and starter code. A
grader, kept outside the task workspace during evaluation, executes behavioral
tests after the agent edits the workspace.

## Core construct
Whether an agent promotes evidence into a stronger authority conclusion than the
evidence supports. The case set distinguishes determinate positive evidence,
determinate negative evidence, underdetermining evidence, and contradictory
evidence. See `docs/CONSTRUCT_VALIDITY.md`.

## Decision models
Authority-decision cases (PF-011, PF-012, PF-013) return one of `GRANT`,
`RETAIN`, `REVOKE`, `QUARANTINE`, defined identically across cases in
`docs/AUTHORITY_DECISION_MODEL.md`. Other cases return a parse result, a
validation error, a boolean fact, or a three-valued fact. Each case declares its
`decision_model` in `case.json`.

## Anti-degeneracy
`QUARANTINE` is not a free answer. Every authority-decision case commits
`always-grant`, `always-retain`, `always-revoke`, and `always-quarantine` mutants
that its grader must kill, alongside near-miss mutants for the specific failure
the case is named after. `go run ./cmd/proof-fence mutation` re-checks all 78
committed mutants. Scoring is all-or-nothing per case; there is no partial credit
for a degenerate strategy.

## Contamination status
Every case is `"exposure": "PUBLIC_PILOT_ONLY"`. Cases, references, and graders
are public and were authored with AI assistance, including by the model family a
later study would evaluate. They cannot support a confirmatory claim about those
models. No held-out confirmatory set exists in this repository.

## Out of scope
Exploit development, penetration testing of third-party systems, malware
generation, production certification, model safety certification, and claims that
success on this benchmark predicts real-world security performance.

## Known limitations
- thirteen public pilot cases; Go only;
- reference solutions are public for reproducibility;
- graders are binary pass/fail and do not score explanation quality;
- a lookup table keyed on the exact fixture tuples passes any finite executable
  grader, these included; only a held-out set excludes that;
- `QUARANTINE` does not distinguish underdetermination from contradiction;
- PF-011 and PF-012 cannot reach `GRANT`: in both, the subject already holds the
  authority being decided;
- PF-013's determinate space is saturated at four corroborated combinations;
- no held-out confirmatory set is included, and none should be added here;
- no human inter-rater reliability study has been run;
- no comparative novelty claim has been established;
- no baseline or model evaluation has been run against v0.2;
- submissions may edit only `challenge.go`; the trusted starter `go.mod` is
  restored, extra candidate files are rejected, and a documented source policy
  (safe imports, no compiler/build/position directives, no test or lifecycle
  declarations, no executable package-level variable initializers) is applied
  before any candidate code runs;
- trusted grader tests are compiled once and then executed one at a time from
  controller-selected names; every transcript line carries an origin label and
  candidate output is never parsed as result authority;
- build and execution have separate finite timeouts, and exceeding either is
  recorded as `INFRASTRUCTURE` rather than as a semantic failure;
- the exact Go toolchain is recorded per run; a case's `go 1.23` directive is a
  language-version floor, not the host toolchain;
- ProofFence does not provide OS-level sandboxing for candidate code; graders
  must run in disposable isolation.

## Provenance
The case taxonomy was abstracted from recurring failure patterns observed during
private defensive cloud-control-plane engineering. No private source code,
credentials, live account identifiers, or raw review reports are included.
