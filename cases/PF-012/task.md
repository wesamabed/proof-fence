# PF-012 — re-attesting a grant at a policy generation

`challenge.go` is the decision step of a continuous-compliance controller.

## The subject of the decision

Every subject in this controller already holds an active grant. Each pass returns
one `Decision` about that grant, and the platform applies it immediately:

- `GRANT` — issue a grant to a subject that has none.
- `RETAIN` — leave the existing grant exactly as it is until the next pass.
- `REVOKE` — remove the subject's grant. The subject loses access at once.
- `QUARANTINE` — suspend the grant, stop automated passes for this subject, and
  open an operator ticket describing the evidence.

## Generations

A subject's configuration is versioned into numbered *generations*. A generation
is an immutable snapshot: generation 7 and generation 6 are two different
configurations, and the platform places no constraint on how much they differ.
Generation numbers start at 1.

A `ChangeRequest` names the subject and the single generation whose configuration
this pass is deciding about:

- `Subject` — the subject identifier. Identifiers are opaque and compared
  literally.
- `Generation` — the generation being decided.

## Observations

An independent compliance checker reads one subject at one generation, evaluates
it against the published requirements, and signs a record of what it read. The
controller loads a record and hands it to you as an `Observation`:

- `Present` — an observation record was loaded.
- `Verified` — the controller checked the record's signature against the
  checker's published key and reports the result here.
- `Subject` — the subject identifier the checker recorded for the configuration
  it read.
- `Generation` — the generation number the checker recorded for the configuration
  it read.
- `Compliant` — whether the configuration the checker read met the published
  requirements.

An observation records the checker's evaluation of the one subject and the one
generation named inside that observation.

The request and the observation arrive from different sources, and the
controller's loader does not cross-check them before calling you.

## Task

Implement `AuthorityDecision` so that each pass yields the decision its evidence
supports.
