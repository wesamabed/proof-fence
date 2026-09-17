# Security policy

ProofFence is a defensive benchmark. Please report vulnerabilities in the
benchmark harness or accidental disclosure of sensitive material privately to the
repository owner before public discussion.

Do not submit:
- credentials, tokens, private keys, or live account identifiers;
- exploit material targeting systems you do not own or have authorization to
  test;
- proprietary source code without permission;
- raw private reports or source code from the motivating private project.

Synthetic cases and fixes to the public benchmark are welcome through normal pull
requests.

## Evaluator execution boundary

`proof-fence grade` executes candidate-authored Go code. Treat that code as
untrusted. Use a disposable VM or container with no cloud credentials, tokens,
SSH agents, sensitive mounts, or host secrets; restrict network access where
practical. ProofFence refuses unsandboxed grading unless
`PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1` is set. That variable is an
acknowledgment, not a sandbox.

## Result-integrity boundary

ProofFence uses a narrow source-edit submission model. Before any candidate code
executes, the controller:

- accepts only a regular `challenge.go` in package `challenge`;
- requires the candidate `go.mod` to be byte-identical to the trusted starter;
- rejects extra files, including candidate `_test.go` files;
- rejects compiler, build, and position directives — `//go:`, `//line`,
  `/*line`, `// +build`, `//export`, `//extern` — because these can change how
  the toolchain reads the source the controller just validated;
- rejects `init`, `TestMain`, and `Test*` / `Benchmark*` / `Fuzz*` declarations;
- rejects package-level variable initializers containing a function call,
  conversion, or function literal, because Go evaluates those during package
  initialization, before any trusted test body runs;
- permits only a small documented set of non-process-control standard-library
  imports.

The controller then reconstructs the module from trusted inputs, overlays the
trusted grader, compiles once, and executes each trusted top-level test
separately from the compiled binary with an exact `-test.run` selector and the
test cache disabled. `PASS` is derived only from the successful execution of
every controller-selected trusted test.

Candidate stdout is never parsed as test authority, so forged `=== RUN` /
`--- PASS` markers cannot create a pass. Every transcript line additionally
carries an origin label, and the controller prepends its label to each subprocess
line, so candidate text that imitates a `[proof-fence CONTROLLER]` line is
recorded as `[proof-fence CANDIDATE_OUTPUT] [proof-fence CONTROLLER] …`. This
protects the human reading the transcript, not just the score.

Compilation and execution carry separate finite, fail-closed budgets. Exceeding
either is recorded as `INFRASTRUCTURE`, distinct from a semantic `FAIL`.

This source policy is a **benchmark result-integrity boundary, not an OS
sandbox**. Candidate `challenge.go` still executes native Go code. Continue to
grade inside a disposable VM or container with no secrets, privileged
credentials, or sensitive mounts, and restrict network access where practical.

## Contamination boundary

Every case in this repository is `PUBLIC_PILOT_ONLY`. Do not add a held-out
confirmatory case to this repository, and do not present a public pilot case as
held-out evidence. A held-out set must be authored separately and kept
permanently private.
