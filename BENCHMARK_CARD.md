# Benchmark card — ProofFence v0.1

## Intended use
Defensive evaluation of coding agents and review workflows on small security-semantic programming tasks.

## Unit of evaluation
A case is a small isolated Go module with a task statement and starter code. A grader, kept outside the task workspace during evaluation, executes behavioral tests after the agent edits the workspace.

## Core construct
Whether lower-trust, ambiguous, contradictory, or unauthenticated evidence is incorrectly promoted into a stronger positive security fact.

## Out of scope
Exploit development, penetration testing of third-party systems, malware generation, production certification, model safety certification, and claims that success on this benchmark predicts real-world security performance.

## Known limitations
- only ten public pilot cases;
- Go-only in v0.1;
- reference solutions are public for reproducibility;
- graders are binary pass/fail and do not yet score explanation quality;
- no held-out confirmatory set is included;
- no human inter-rater reliability study has been run;
- no comparative novelty claim has been established;
- v0.1 accepts only candidate edits to `challenge.go`, restores the trusted starter `go.mod`, rejects extra candidate files, and applies a documented safe-import/source policy before trusted tests run;
- trusted grader tests are executed one at a time from controller-selected names; candidate stdout is diagnostic only and is never parsed as result authority;
- v0.1 does not provide OS-level sandboxing for candidate code; graders must run in disposable isolation.

## Provenance
The case taxonomy was abstracted from recurring failure patterns observed during private defensive cloud-control-plane engineering. No private source code, credentials, live account identifiers, or raw review reports are included.
