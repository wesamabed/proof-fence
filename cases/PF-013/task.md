# PF-013 — rotating a signing key

`challenge.go` is the decision step of a signing-key rotation workflow.

## The subject of the decision

A *slot* is one named position in the trust store. Each rotation pass returns one
`Decision` about the slot's signing authority, and the platform applies it
immediately:

- `GRANT` — publish the slot's key to the trust store. Downstream verifiers begin
  accepting signatures made with it.
- `RETAIN` — leave the slot's trust-store entry exactly as it is.
- `REVOKE` — remove the slot's entry from the trust store. Signatures made with
  that key stop verifying everywhere.
- `QUARANTINE` — mark the slot inconsistent, stop automated rotation passes for
  it, and open an operator ticket describing the evidence.

## The request

A `Request` names one `Intent`:

- `INSTALL` — write a new key into the slot. The workflow issues `INSTALL` only
  for a slot that currently holds no key.
- `REMOVE` — erase the key from the slot. The workflow issues `REMOVE` only for a
  slot that currently holds a key.

A value outside this set comes from a workflow version this client release does
not model, and neither precondition above applies to it.

## Evidence modality 1: the service's own report

`Apply` sends the request to the key service, which returns an `ApplyReport`:

- `Returned` — a response arrived. When it is false the transport gave up and
  there is no report; the other field is not populated.
- `Status` — one of:
  - `APPLIED` — the service reports it carried out the request.
  - `REJECTED` — the service reports it declined the request and changed nothing.
  - `ERROR` — the service could not determine or report what happened. The
    request may or may not have taken effect.

  A value outside this set comes from a service version this client release does
  not model.

The service is reporting on its own action, from its own records; it does not
read the slot. Read together with the preconditions above, a report that the
service carried out or declined the request amounts to a claim about what the
slot now holds. But an account a system gives of its own action does not
establish what is actually stored there.

## Evidence modality 2: an independent readback

A separate reader, holding its own credential and talking to the storage layer
rather than to the key service, reads the slot directly and signs what it read.
The workflow loads that record as a `Readback`:

- `Present` — a readback record was loaded.
- `Verified` — the workflow checked the record's signature against the reader's
  published key and reports the result here.
- `State` — what the reader found in the slot:
  - `KEY_PRESENT` — the slot holds a key.
  - `KEY_ABSENT` — the slot holds no key.
  - `UNREADABLE` — the reader reached the slot but could not decode its contents.

The two modalities are produced by different systems from different vantage
points and are handed to your function without being reconciled.

## Task

Implement `RotationDecision` so that each pass yields the decision its evidence
supports.
