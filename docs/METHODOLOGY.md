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

`materialize` copies only the starter workspace and task statement to an isolated directory. The coding agent should receive only that directory. `grade` copies the public grader into a temporary copy of the candidate workspace and runs `go test ./...`.

This separation prevents the normal evaluation path from handing the grader to the agent, even though the public repository remains fully reproducible.

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
