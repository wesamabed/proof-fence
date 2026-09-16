# Security policy

ProofFence is a defensive benchmark. Please report vulnerabilities in the benchmark harness or accidental disclosure of sensitive material privately to the repository owner before public discussion.

Do not submit:
- credentials, tokens, private keys, or live account identifiers;
- exploit material targeting systems you do not own or have authorization to test;
- proprietary source code without permission;
- raw private Arbiter reports or source code.

Synthetic cases and fixes to the public benchmark are welcome through normal pull requests.

## Evaluator execution boundary

`proof-fence grade` executes candidate-authored Go code. Treat that code as untrusted. Use a disposable VM/container with no cloud credentials, tokens, SSH agents, sensitive mounts, or host secrets; restrict network access where practical. v0.1 intentionally refuses unsandboxed grading unless `PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1` is set. That variable is an acknowledgment, not a sandbox.

## Result-integrity boundary

ProofFence v0.1 deliberately uses a narrow source-edit submission model. Before any candidate code executes, the controller accepts only a regular `challenge.go`, requires the candidate `go.mod` to remain byte-identical to the trusted starter, rejects extra files (including candidate `_test.go` files), rejects compiler directives and test/lifecycle declarations, and permits only a small documented set of non-process-control standard-library imports. The controller then reconstructs the module from trusted inputs and overlays the trusted grader.

Each trusted top-level grader test is selected and executed separately with `go test -count=1 -run '^<trusted-name>$'`. PASS is derived only from successful execution of every controller-selected trusted test. Candidate stdout is never parsed as test authority, so forged `=== RUN` / `--- PASS` markers cannot create a pass.

This source policy is a **benchmark result-integrity boundary**, not an OS sandbox. Candidate `challenge.go` still executes native Go code. Continue to grade inside a disposable VM/container with no secrets, privileged credentials, or sensitive mounts and restrict network access where practical.
