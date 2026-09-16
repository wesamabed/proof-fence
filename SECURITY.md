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

The grader does not trust candidate stdout or a zero test-process exit on its own. It runs `go test -json .` and independently requires every trusted top-level grader test to emit both `run` and `pass` events without `fail` or `skip`. This blocks early-success `TestMain` suppression of the trusted assertions. It does not make same-process Go execution a sandbox; disposable external isolation remains mandatory for untrusted candidates.
