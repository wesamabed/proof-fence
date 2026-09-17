# Methodology

## Construct

ProofFence models security decisions as a promotion from evidence to authority.
A candidate implementation is unsafe when it reaches a conclusion the available
evidence does not carry — either by treating a lower-quality input as a stronger
fact, or by resolving an underdetermined state as though it were determined.

`docs/CONSTRUCT_VALIDITY.md` states what is measured, why v0.1 could not measure
it, and the per-case review that v0.2's repair was checked against.

## Dimensions

1. **Admission** — exact, complete parsing of security-bearing data.
2. **Provenance** — the fact came from the required producer, about the required
   subject, at the required anchor.
3. **Coherence** — outcome class, fault state, response body, and independent
   readings agree.
4. **Closure** — all required premises exist before a positive result is emitted.
5. **Execution proof** — security tests were independently observed to run and
   pass.
6. **Authority decision** — the four-valued transition itself, under evidence
   that may be incomplete, stale, or contradictory.

The first five produce facts. The sixth consumes them. See
`docs/AUTHORITY_DECISION_MODEL.md`.

## Writing a case that measures reasoning

A ProofFence task statement specifies the API and data semantics, the operational
goal, what each field means and which component populates it, the relevant trust
properties as facts about origin and attribution, and what the caller does with
the result.

It does not state the boolean conjunction, the branch mapping, what to return
when a field is missing, which source is "trusted" as a verdict, or the reference
solution in prose. A candidate that transcribes one sentence of the task into Go
must fail.

The ambiguity belongs in the evidence, never in the instructions. Every case
remains well-specified: one deterministic correct answer per input.

Identifiers name origins, not trust levels. `SelfReportPassed` beside
`SupervisorPassed` is legitimate — knowing which origin establishes a fact is the
reasoning under test. `TrustedObservedPass` beside `ClaimedPass` is not.

## Evaluation workflow

`materialize` copies the trusted starter workspace and prepends the submission
contract to `TASK.md`. The coding agent receives only that directory.

ProofFence is deliberately a **source-edit benchmark**: only `challenge.go` may
change. During grading the controller rejects extra candidate files, requires
`go.mod` to match the trusted starter byte-for-byte, checks that `challenge.go`
is a regular Go source file in package `challenge`, rejects compiler, build, and
position directives, rejects candidate test and lifecycle declarations, rejects
package-level variable initializers that contain calls or function literals, and
permits only a small allowlist of standard-library imports that excludes process
and test lifecycle control.

The package-level-variable rule closes the pre-test execution class. Go evaluates
package-level initializers during package initialization — before any trusted
test body runs. Rather than reason about which pre-test execution is harmless,
the policy removes the mechanism. It is deliberately conservative: a type
conversion parses as a call and is rejected too, and the equivalent typed
declaration (`var x Kind = "A"`) is always available. This is a mechanical,
documented rule, not a malicious-Go analyzer.

The controller then reconstructs a fresh temporary module from the trusted
`go.mod`, the validated candidate `challenge.go`, and the trusted grader.

## Execution and result integrity

The controller derives the trusted top-level grader-test inventory from grader
source. It compiles the reconstructed module once into a test binary, then
executes each trusted test separately from that binary with an exact `-test.run`
selector and the test cache disabled. `PASS` requires every controller-selected
test to succeed.

Compilation and execution carry separate finite budgets
(`PROOF_FENCE_BUILD_TIMEOUT`, default 3m; `PROOF_FENCE_TEST_TIMEOUT`, default
1m). Separating them stops a cold compile from consuming the execution budget and
producing a spurious model `FAIL`. Both fail closed: exceeding either never
yields `PASS`.

Every transcript line carries an origin label:

- `[proof-fence CONTROLLER]` — the benchmark's own statements;
- `[proof-fence TRUSTED_TEST]` — a controller-derived verdict for one named
  trusted test, taken from process exit status;
- `[proof-fence CANDIDATE_OUTPUT]` — raw subprocess bytes.

Subprocess output is an inseparable mixture of trusted testing-framework output
and arbitrary candidate stdout, so the whole block is labelled as candidate
output and carries no authority. Because the controller prepends its label to
every subprocess line, a candidate that prints a controller-shaped line has it
recorded as `[proof-fence CANDIDATE_OUTPUT] [proof-fence CONTROLLER] …`. The
label always wins.

This is a **result-integrity boundary, not an OS sandbox**. Candidate
`challenge.go` still executes native Go code after the source-policy gate.
Reproducible studies must grade inside a disposable VM or container with no
secrets, privileged credentials, or sensitive mounts, and should restrict network
access where practical. The environment opt-in acknowledges that requirement; it
does not provide isolation.

## Scoring

One primary outcome per trial:

- `PASS` — every case-specific property holds under the grader;
- `FAIL` — at least one property does not hold, or compilation fails, or the
  submission violates the source policy;
- `INFRASTRUCTURE` — a build or execution timeout, or a toolchain failure.

`INFRASTRUCTURE` is not a model failure. A study should drop or re-run those
trials and report how many there were.

Studies should additionally record: the exact Go toolchain
(`proof-fence toolchain`), wall-clock time, model and tool configuration, agent
turns or tool calls where available, patch size, the agent's stated rationale,
and reviewer findings with whether the grader confirms them.

## Self-test and mutation

`proof-fence selftest` proves, per case, that the starter fails **semantically**
and the reference passes. A starter that fails for infrastructure reasons is
itself an error.

`proof-fence mutation` runs every committed mutant under
`cases/<ID>/mutants/`. Each is a near-miss or degenerate-strategy implementation
that the grader must kill. A survivor means the grader does not discriminate the
behaviour the case claims to measure. The same suite runs under `go test ./...`.

### What a mutation score establishes, and what it does not

A killed mutant establishes that **this grader is sensitive to this mutant**. It
does not establish that the grader is **correct**, and the suite's own structure
is why.

The suite's pass criterion is `survivors = 0`. Every committed mutant is
therefore, by construction, something the grader is required to kill. So if a
**defensible alternative reading** of a task — an implementation a competent
reader could justify from the published text — is committed as a mutant, the
suite records its rejection as a *discrimination success*. Mutation testing
applied to a benchmark grader does not merely fail to detect oracle ambiguity:
it converts oracle ambiguity into a reported quality signal.

This is not hypothetical here. ProofFence v0.2 scored **78 killed / 0
survivors**, and an independent blind audit of the same artifact then found 20 of
its 138 fixtures underspecified — with **four committed mutants being exactly the
defensible alternative that produced their case's contested fixture**. The full
finding is in [`V0.2_BLIND_ORACLE_AUDIT.md`](V0.2_BLIND_ORACLE_AUDIT.md).

Two consequences are built into v0.3:

1. **Every mutant carries a recorded kind.** `cases/<ID>/mutants/classification.json`
   labels each committed mutant `degenerate-strategy`, `mutation`, or
   `alternative-reading-probe`, and `proof-fence mutation` prints the kind beside
   every result and the totals by kind. The classification is **adjudicated data,
   not a computed property**: deciding whether an implementation is a defensible
   reading of a task requires reading the task and judging what its text entails,
   and no classifier in this repository attempts that. A mutant with no recorded
   kind fails the suite.
2. **The caveat travels with the number.** `proof-fence mutation` prints, on every
   run, that a killed mutant shows sensitivity rather than correctness and that
   oracle validity is established by independent specification audit. Any
   reporting of a ProofFence mutation score must carry the same qualification.

## Exact-delta review condition

A review study can compare:

1. implementation only;
2. implementation + same-agent self-review;
3. implementation + independent reviewer;
4. implementation + independent reviewer + one focused correction + exact-delta
   verification.

The benchmark does not claim which is superior. That is an empirical question.

## Public versus held-out cases

Every case here is a public pilot case, marked `"exposure":
"PUBLIC_PILOT_ONLY"`. They were authored with AI assistance, including by the
model family a later study would evaluate, so they are contaminated for a clean
confirmatory measurement of those models.

Confirmatory research must build a separately authored, permanently private
held-out set from this taxonomy and preregister its scoring before any model
runs. No held-out set exists in this repository, and none should be added here.

## Reviewer independence for confirmatory studies

Before a confirmatory comparison, preregister what "independent reviewer" means:
whether implementation transcripts or model identity are visible, whether
reviewer context is freshly instantiated, the allowed artifacts, model and
version allocation, contamination handling, stopping and exclusion rules, and the
primary estimand.
