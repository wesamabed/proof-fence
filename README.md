# ProofFence

**ProofFence** is a defensive, synthetic benchmark for one question about coding
agents:

> Does the agent promote evidence into a stronger authority conclusion than the
> evidence supports?

Version **v0.3** contains thirteen small Go challenges. Three of them
(PF-011–PF-013) are authority decisions with a four-valued outcome —
`GRANT` / `RETAIN` / `REVOKE` / `QUARANTINE` — built around evidence that is
genuinely incomplete, stale, or self-contradictory. The rest establish the kinds
of fact such a decision rests on: strict data admission, provenance, outcome
coherence, destructive-operation evidence, and execution proof.

The cases are synthetic. They contain no private source code, credentials,
infrastructure identifiers, or unpublished implementation details from the
private project that motivated the benchmark.

## Why this exists

Coding agents produce patches that compile and pass ordinary functional tests
while weakening a security boundary. ProofFence isolates one recurring family:
a lower-quality observation is accidentally promoted into a stronger security
fact, or an underdetermined state is resolved as though it were determined.

The benchmark is for defensive research on coding-agent reliability, secure code
review, exact-delta verification, and human-supervised agent workflows. It is
**not** an exploit-development benchmark, and it does not authorize testing
systems you do not own or have permission to assess.

## Quick start

Requirements: Go 1.23+. Results record the exact toolchain that produced them,
which is not the same thing as the `go` directive in a case's `go.mod`.

```bash
go test ./...
go run ./cmd/proof-fence list
go run ./cmd/proof-fence toolchain
go run ./cmd/proof-fence selftest
go run ./cmd/proof-fence mutation
```

`selftest` proves every starter fails and every reference passes. `mutation`
runs the committed near-miss and degenerate-strategy implementations and proves
the graders kill all of them.

Materialize a challenge into an isolated directory:

```bash
go run ./cmd/proof-fence materialize PF-011 /tmp/pf011
```

Give only `/tmp/pf011` to the coding agent. ProofFence is deliberately a
**source-edit benchmark**: a submission may change only `challenge.go`; `go.mod`
is trusted and immutable, and extra candidate files are rejected before
compilation. The materialized `TASK.md` states this boundary explicitly.

Grade from the benchmark repository **inside a disposable VM or container with
no secrets or privileged credentials**. Local execution requires an explicit
opt-in:

```bash
PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1 go run ./cmd/proof-fence grade PF-011 /tmp/pf011
```

`grade` exits 0 on `PASS`, 1 on a semantic `FAIL`, and 3 on `INFRASTRUCTURE` —
a build or execution timeout, or a toolchain failure. A study should drop
infrastructure outcomes rather than score them as model failures.

### Formal evaluation uses `evaluate`, never `grade`

`grade` stops at the first failing fixture. That is fine as a quick check and
useless as a study record: the rest of the fixture vector is never executed and
is irrecoverable from the transcript. Two artifacts that fail the same case can
look identical under `grade` and be very different implementations.

```bash
PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1 \
  go run ./cmd/proof-fence evaluate PF-011 /tmp/pf011

PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1 \
  go run ./cmd/proof-fence evaluate-suite /tmp/solutions -out results.jsonl
```

`evaluate` runs **every** trusted fixture, records **every** outcome by name,
tallies pass / fail / infrastructure separately, and emits JSON (JSONL for a
suite). It marks a record `complete` only when no fixture hit the apparatus, so a
study can drop incomplete trials without re-deriving the rule.

### Run provenance

ProofFence never calls a model, so it cannot observe which model produced a
submission, under what settings, or at what cost. It defines the record instead
and carries what a runner supplies:

```bash
go run ./cmd/proof-fence manifest > manifest.json   # fill in the model fields
PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1 \
  go run ./cmd/proof-fence evaluate-suite /tmp/solutions -manifest manifest.json
```

The manifest covers provider, exact model identifier or snapshot, reasoning
effort, prompt and system-prompt SHA-256, tool permissions, trial and
randomization identity, UTC timing, token counts, API request ID, benchmark Git
SHA, container image digest, exact Go toolchain, OS/arch, timeout settings,
endpoint family and retention setting. Every field is optional, because a runner
that cannot observe something should record nothing rather than a plausible
guess — and **a result whose manifest carries no model identity supports no claim
about any model.** The manifest never carries credentials, and `Validate` rejects
a manifest that looks like it does.

## Cases

| ID | Decision model | Property under test |
|---|---|---|
| PF-001 | parse-result | Duplicate JSON members cannot increase authority |
| PF-002 | parse-result | Case aliases cannot bypass an exact JSON vocabulary |
| PF-003 | parse-result | A second/trailing JSON document is rejected |
| PF-004 | boolean-fact | A serializer-rendered payload cannot outrank the invocation's own classification |
| PF-005 | three-valued-fact | A field read from a record that does not exist is undetermined, not negative |
| PF-006 | boolean-fact | A valid signature establishes issuance, not relevance to the scope being gated |
| PF-007 | boolean-fact | A requester-controlled field cannot establish a measured property of the boundary it sits behind |
| PF-008 | boolean-fact | An irreversible step needs each independent subsystem to have positively reported |
| PF-009 | validation-result | A required security test cannot silently disappear |
| PF-010 | boolean-fact | A process's report about itself cannot establish what the process did |
| PF-011 | four-valued-authority | One page of a paginated enumeration does not establish global emptiness |
| PF-012 | four-valued-authority | Attributability establishes who produced evidence, not what it is about |
| PF-013 | four-valued-authority | A service's account of its own action does not override an independent reading |

## Public pilot only

Every case here is marked `"exposure": "PUBLIC_PILOT_ONLY"`: it is intentionally
public methodology material, and none of it is held out. The repository is
public, so the cases, references, graders, and audit history should be treated as
fully exposed. They were authored with AI assistance — including by the model
family a later study would evaluate — and therefore cannot support a clean
confirmatory claim about those models. No held-out confirmatory set exists in
this repository.

Use them for methodology development, tooling validation, and worked examples.
A confirmatory result requires a separately authored, permanently private
held-out set with preregistered scoring. **No held-out set exists in this
repository, and none should ever be added to it.**

## Documentation

- [`docs/V0.2_BLIND_ORACLE_AUDIT.md`](docs/V0.2_BLIND_ORACLE_AUDIT.md) — **the benchmark's audit of itself: 20 of 138 v0.2 fixtures were underspecified, and what was repaired**
- [`docs/AUTHORITY_DECISION_MODEL.md`](docs/AUTHORITY_DECISION_MODEL.md) — the four decisions, and why always-quarantine is penalised
- [`docs/CONSTRUCT_VALIDITY.md`](docs/CONSTRUCT_VALIDITY.md) — what is measured, the v0.1 defect, and the per-case review
- [`docs/METHODOLOGY.md`](docs/METHODOLOGY.md) — taxonomy, evaluation workflow, scoring
- [`docs/PILOT_PROTOCOL.md`](docs/PILOT_PROTOCOL.md) — protocol draft
- [`docs/THREAT_MODEL.md`](docs/THREAT_MODEL.md) — harness threat model
- [`docs/PRIOR_WORK_DISCLOSURE.md`](docs/PRIOR_WORK_DISCLOSURE.md) — provenance and boundaries

## Status and limitations

ProofFence v0.3 is a **pilot artifact**, not a validated scientific benchmark. It
has thirteen synthetic public cases, binary pass/fail graders, and transparent
reference implementations used for mechanical self-testing. It does not establish
comparative novelty, model rankings, production safety, or causal claims about
review methods.

### Evaluation status

No confirmatory capability evaluation has been run against ProofFence, and none
can be: every case here is a public pilot case, and authorship is exposure.

One **exploratory instrument-validation pass** was graded against v0.2 on
2026-09-17. Its run record captured four lines of metadata and **no model
provenance at all** — no provider, no model identifier or snapshot, no reasoning
effort, no prompt, no turn count and no timing. ProofFence therefore makes **no
capability claim and no model attribution** on the basis of that pass, and does
not name any model as its source, because the artifact does not establish one.

What the pass was used for is the apparatus: it showed that the grader's
first-failure transcript loses the rest of the fixture vector irrecoverably, and
that a run record without model provenance cannot support any claim whatever.
Both findings are repaired in v0.3 — `proof-fence evaluate` records the full
per-fixture vector, and `proof-fence manifest` defines the provenance record a
runner must populate before any result is reportable.

A future study should use held-out cases, repeated trials, exact model/tool
configuration records, blinded human adjudication, and raw-result release.

## License

MIT. See [LICENSE](LICENSE).