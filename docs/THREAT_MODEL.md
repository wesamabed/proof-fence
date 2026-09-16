# Threat model for the benchmark harness

## Assets
- integrity of case metadata and graders;
- separation between agent task workspace and grader;
- reproducibility of results;
- privacy boundary between synthetic public cases and motivating private work.

## Threats
- task process reads grader/reference files because evaluator fails to isolate workspaces;
- candidate code forges textual "test passed" output;
- benchmark accepts compilation or process exit without executing required assertions;
- case content accidentally contains private identifiers or unpublished source;
- evaluator changes model/tool configuration without recording it;
- public pilot cases are presented as held-out evidence;
- candidate code escapes the evaluation workspace or reads host secrets/network resources;
- candidate code introspects grader artifacts at runtime.

## Controls
- materialize only starter files into the task workspace;
- grade in a fresh temporary copy;
- use trusted grader files supplied by the benchmark controller;
- treat candidate-authored claims as untrusted;
- keep public and held-out sets explicitly separate;
- sanitize all case identities and fixtures;
- run grading only in disposable isolation with no secrets/privileged credentials and restricted network access;
- treat v0.1 grader-source separation as pre-execution hiding, not as a malicious-candidate security boundary.
