# Methodology

## Security invariant

ProofFence models security decisions as a promotion from evidence to authority. A candidate implementation is unsafe when a lower-quality input can create a positive security fact that should require stronger evidence.

The v0.1 cases exercise five recurring dimensions:

1. **Admission** — exact, complete parsing of security-bearing data.
2. **Provenance** — the fact came from the required producer/context.
3. **Coherence** — outcome class, error state, and response body agree.
4. **Closure** — all required premises exist before a positive result is emitted.
5. **Execution proof** — security tests were independently observed to run and pass.

## Evaluation workflow

`materialize` copies the trusted starter workspace and prepends a common submission contract to `TASK.md`. The coding agent receives only that directory. v0.1 is intentionally a **source-edit benchmark**: only `challenge.go` may change. During grading, ProofFence rejects extra candidate files, requires `go.mod` to match the trusted starter byte-for-byte, checks that `challenge.go` is a regular Go source file in package `challenge`, rejects compiler directives and candidate test/lifecycle functions, and permits only a small allowlist of standard-library imports that excludes process/test lifecycle control. The controller then creates a fresh temporary module from the trusted `go.mod`, the validated candidate `challenge.go`, and the trusted grader.

The controller derives the trusted top-level grader-test inventory from grader source and executes each trusted test separately with an exact `-run` selector and test-cache disabled. Overall `PASS` requires every controller-selected test command to succeed. Candidate stdout is diagnostic only; ProofFence does not parse candidate-authored `=== RUN`, `--- PASS`, or other textual markers as authority. This closes the public-pilot `TestMain`/forged-marker result-integrity class by removing candidate test files and lifecycle/process-control mechanisms from the accepted submission surface.

This is still **not** a hostile-code sandbox. Candidate `challenge.go` executes native Go code after the source-policy gate, so reproducible studies must grade inside a disposable VM/container with no secrets, privileged credentials, or sensitive mounts and should restrict network access where practical. The environment opt-in acknowledges that external isolation requirement; it does not provide isolation itself.

## Scoring

v0.1 produces one primary metric per trial:

- `PASS`: all case-specific security properties hold under the grader.
- `FAIL`: at least one property does not hold, compilation fails, or the test process fails.

Research studies should additionally record:
- wall-clock time;
- model/tool configuration;
- number of agent turns/tool calls if available;
- patch size;
- explanation/rationale;
- reviewer findings and whether they are confirmed by the grader.

## Exact-delta review condition

A review study can compare:
1. implementation only;
2. implementation + same-agent self-review;
3. implementation + independent reviewer;
4. implementation + independent reviewer + one focused correction + exact-delta verification.

The benchmark itself does not claim which condition is superior. That is an empirical question.

## Public vs held-out cases

The ten v0.1 cases are public pilot cases. Confirmatory research should create a separately held-out set from the same published taxonomy and preregister its scoring before model runs.

## Reviewer independence for confirmatory studies

The public pilot does not make causal claims about review workflows. Before a confirmatory comparison, preregister what “independent reviewer” means: whether implementation transcripts or model identity are visible, whether reviewer context is freshly instantiated, the allowed artifacts, model/version allocation, contamination handling, stopping/exclusion rules, and the primary estimand.
