# Pilot evaluation protocol draft

This is a protocol draft, not a completed or preregistered study. **No baseline
or model evaluation has been run against ProofFence v0.2.**

## Contamination boundary

Every case in this repository is `"exposure": "PUBLIC_PILOT_ONLY"`. The cases,
references, and graders are public and were authored with AI assistance,
including by the model family a later study would evaluate.

A pilot run against these cases can validate tooling, timing, prompts, and
scoring mechanics. It cannot support a capability claim about a model that may
have seen them. Any confirmatory comparison needs a separately authored,
permanently private held-out set with scoring preregistered before model runs.

## Primary questions
1. Does independent review improve the rate at which security-semantic defects
   are detected and repaired?
2. Does exact-delta verification detect residual defects after a focused
   correction?
3. Do model reviewers over-report findings that are not load-bearing under
   executable graders?
4. Under evidentiary ambiguity, do agents return a determinate authority
   conclusion the evidence does not support — and conversely, do they quarantine
   states that are in fact determinate?

## Suggested pilot
- cases: PF-001 through PF-013;
- implementation trials: at least 5 per model/configuration;
- conditions: implementation-only, self-review, independent-review, and the
  exact-delta workflow;
- randomize case order;
- preserve exact prompts, model identifiers, tool permissions, and timestamps;
- grade automatically before human interpretation;
- preserve the source-edit submission boundary across all compared conditions;
- execute every candidate in a fresh disposable VM or container with no secrets
  or privileged credentials, and restrict network access where practical;
- blind human adjudicators to model identity when scoring explanations and
  reviewer findings.

## Primary outcomes
- executable pass rate;
- confirmed security-defect detection rate;
- residual-defect rate after correction;
- false-positive reviewer finding rate.

## Authority-decision outcomes

For PF-011, PF-012, and PF-013, record the returned decision per fixture class,
not only pass/fail. Two error directions matter and should be reported
separately:

- **over-promotion** — a determinate conclusion (`GRANT`, `RETAIN`, `REVOKE`)
  returned where the evidence is underdetermining or contradictory;
- **over-refusal** — `QUARANTINE` returned where the evidence is determinate.

Reporting only an aggregate pass rate hides which direction an agent fails in,
and the two have different operational costs.

## Secondary outcomes
- patch size;
- runtime and usage where available;
- reason correctness;
- reviewer disagreement.

## Handling infrastructure outcomes

`proof-fence grade` exits 3 and reports `INFRASTRUCTURE` for a build or execution
timeout or a toolchain failure. These are apparatus failures, not model failures.
Preregister whether such trials are re-run or dropped, and report how many
occurred. Do not fold them into the failure rate.

## Reproducibility metadata

For any formal comparative run, record the exact Go toolchain as reported by
`proof-fence toolchain` (a case's `go 1.23` directive is a language-version
floor, not the host toolchain), the OS and architecture, the container or VM
image digest, the configured `PROOF_FENCE_BUILD_TIMEOUT` and
`PROOF_FENCE_TEST_TIMEOUT`, and exact GitHub Action commit SHAs if GitHub Actions
is part of the measured environment. The pilot CI may use maintained major action
tags; confirmatory results should pin exact environment identities.

## Reporting rule

Publish negative results, reviewer overreach, and failed corrections as well as
successes. Do not infer production security from benchmark performance. State the
contamination status of every case set used.
