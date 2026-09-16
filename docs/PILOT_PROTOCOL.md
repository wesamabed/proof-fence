# Pilot evaluation protocol draft

This is a protocol draft, not a completed or preregistered study.

## Primary questions
1. Does independent review improve the rate at which security-semantic defects are detected and repaired?
2. Does exact-delta verification detect residual defects after a focused correction?
3. Do model reviewers over-report findings that are not load-bearing under executable graders?

## Suggested pilot
- cases: PF-001 through PF-010;
- implementation trials: at least 5 per model/configuration;
- conditions: implementation-only, self-review, independent-review, and exact-delta workflow;
- randomize case order;
- preserve exact prompts, model identifiers, tool permissions, and timestamps;
- grade automatically before human interpretation;
- execute every candidate in a fresh disposable VM/container with no secrets or privileged credentials and restricted network access where practical;
- blind human adjudicators to model identity when scoring explanations/reviewer findings.

## Primary outcomes
- executable pass rate;
- confirmed security-defect detection rate;
- residual-defect rate after correction;
- false-positive reviewer finding rate.

## Secondary outcomes
- patch size;
- runtime/usage where available;
- reason correctness;
- reviewer disagreement.

## Reporting rule
Publish negative results, reviewer overreach, and failed corrections as well as successes. Do not infer production security from benchmark performance.
