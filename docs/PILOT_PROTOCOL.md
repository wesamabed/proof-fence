# Pilot evaluation protocol draft

This is a protocol draft, not a completed or preregistered study. **No
confirmatory capability evaluation has been run against ProofFence.**

One exploratory instrument-validation pass was graded against v0.2 on 2026-09-17.
Its run record carried no model provenance — no provider, model identifier,
reasoning effort, prompt, turn count or timing — so **no capability or model
attribution is made from it**, and this document names no model as its source.
The pass is cited only for what it established about the apparatus: that a
first-failure transcript destroys the rest of the fixture vector, and that a run
record must be specified before it is collected, not after. The requirements in
this protocol were tightened in response, and v0.3 enforces them in
`proof-fence evaluate` and `proof-fence manifest`.

## Contamination boundary

Every case in this repository is `"exposure": "PUBLIC_PILOT_ONLY"` — intended
for release, with nothing held out; the repository itself is private and
unreleased. The cases, references, and graders were authored with AI assistance,
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

## Mandatory: record the full per-fixture vector

**Every trial must be recorded with `proof-fence evaluate` or
`proof-fence evaluate-suite`, for every case, not only the four-valued ones.**
`proof-fence grade` stops at the first failing fixture, so its transcript is a
prefix of the truth and the rest of the vector cannot be recovered afterwards.
The 2026-09-17 instrument-validation pass demonstrated the cost directly: the
same artifacts yielded one admissible failure when read from the grade logs and
three when re-evaluated across the full vector.

An `evaluate` record carries every fixture by name with its own verdict, the
pass / fail / infrastructure tallies, and a `complete` flag that is false when
any fixture hit the apparatus. Report per-fixture results alongside the
all-or-nothing case verdict. A single contested fixture flips a whole case under
all-or-nothing scoring, so a case-level number alone can turn one disputed
fixture into an apparent total failure.

## Mandatory: record run provenance before the run, not after

Generate the manifest with `proof-fence manifest`, fill in the model fields, and
pass it to `evaluate`/`evaluate-suite`. A result whose manifest carries no
provider and no exact model identifier supports **no claim about any model**,
regardless of what the directory it was found in is called. Leave a field empty
when the runner cannot observe it; an empty field is an honest record where a
default would be a fabricated one.

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
