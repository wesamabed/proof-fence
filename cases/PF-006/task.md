# PF-006 — attestation scope

`challenge.go` is the admission check in a release gate.

## The API

An attestation service signs statements about build artifacts. Each attestation
covers exactly one scope and exactly one subject, and the issuer records both
inside the signed payload:

- `SignatureValid` — the gate verified the attestation's signature against the
  attestation service's published key and reports the result here.
- `Scope` — the scope identifier the issuer wrote into the payload. Scopes are
  opaque strings compared literally. The service issues attestations across many
  scopes, and a client receives whichever attestations its fetcher pulled, not
  only the ones it asked for.
- `SubjectDigest` — the digest of the artifact the issuer inspected. The issuer
  leaves it empty when it signed a statement it did not bind to any artifact.

`requiredScope` is read from this gate's policy file: the scope this gate is
defined to need evidence for. A gate whose policy names no scope yields an empty
string here.

The empty string is not a scope identifier. The attestation service issues no
attestation under it, and a gate whose policy names no scope has not declared
what evidence it needs.

## What the helper is for

`AttestationSatisfies(requiredScope, a)` decides whether the gate may treat this
attestation as the evidence its policy demands. A `true` result promotes the
artifact to production immediately and runs no further checks; a `false` result
holds the artifact and pages the release engineer.

## Task

Implement `AttestationSatisfies`.
