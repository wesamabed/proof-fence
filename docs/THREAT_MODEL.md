# Threat model for the benchmark harness

## Assets
- integrity of case metadata and graders;
- separation between the agent task workspace and the grader;
- reproducibility of results;
- the privacy boundary between synthetic public cases and the motivating private
  work;
- the contamination boundary between public pilot cases and any future held-out
  confirmatory set.

## Threats
- the task process reads grader or reference files because the evaluator failed
  to isolate workspaces;
- candidate code forges textual "test passed" output;
- candidate code prints controller-shaped lines that mislead a human reading the
  transcript, even where scoring is unaffected;
- candidate code executes before any trusted test body, via a package-level
  variable initializer or a lifecycle declaration;
- candidate source carries a compiler, build, or position directive that changes
  how the toolchain reads the source the controller validated;
- the benchmark accepts compilation or a process exit without executing the
  required assertions;
- a cold compile consumes the execution budget and a correct submission is
  recorded as a model failure;
- results are reported without the toolchain that produced them, and a `go.mod`
  language-version floor is mistaken for the host toolchain;
- case content accidentally contains private identifiers or unpublished source;
- the evaluator changes model or tool configuration without recording it;
- public pilot cases are presented as held-out evidence;
- a held-out confirmatory case is added to this repository and silently
  contaminated;
- candidate code escapes the evaluation workspace or reads host secrets or
  network resources;
- candidate code introspects grader artifacts at runtime.

## Controls
- materialize only starter files into the task workspace;
- grade in a fresh temporary module rebuilt from trusted inputs;
- use trusted grader files supplied by the benchmark controller, and derive the
  test inventory from grader source;
- derive every verdict from process exit status, never from subprocess text;
- label every transcript line with its origin, prepending the label to each
  subprocess line so candidate text cannot impersonate the controller;
- reject the pre-test execution mechanism outright rather than analysing it:
  no `init`, no test or lifecycle declarations, and no package-level variable
  initializer containing a call or function literal;
- reject `//go:`, `//line`, `/*line`, `// +build`, `//export`, and `//extern`
  forms;
- budget compilation and execution separately, keep both finite and
  fail-closed, and record a timeout as `INFRASTRUCTURE` rather than as a
  semantic failure;
- record the exact Go toolchain identity in every self-test, mutation, and grade
  transcript;
- treat candidate-authored claims as untrusted;
- keep public and held-out sets explicitly separate, mark every case's exposure
  in its metadata, and fail the harness if any case is not marked
  `PUBLIC_PILOT_ONLY`;
- sanitize all case identities and fixtures;
- run grading only in disposable isolation with no secrets or privileged
  credentials, and restrict network access where practical;
- treat grader-source separation as pre-execution hiding, not as a
  malicious-candidate security boundary.

## Explicitly not controlled
- OS-level sandboxing of candidate code. Candidate `challenge.go` runs native Go
  after the source-policy gate. External isolation remains the evaluator's job.
- Fixture memorisation. A lookup table keyed on the exact fixture tuples passes
  any finite executable grader. Only a held-out set excludes it.
