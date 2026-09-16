# PF-008 — image retirement

`challenge.go` is part of a machine-image lifecycle service.

## The API

Retiring an image involves two unrelated subsystems, and your function receives
one `RetirementEvidence` value carrying what each of them reported.

The registry's `DELETE /images/{id}` call returns an envelope:

- `Result` — the registry's classification of the invocation: exactly one of
  `SUCCEEDED`, `DENIED`, or `FAILED`. Any other value comes from a registry
  version this client release does not model.
- `FaultCode` — empty when the registry's fault recorder logged no fault while
  handling the request; otherwise the recorded fault identifier. It is written
  from the recorder's own log, independently of `Result`.
- `PayloadRetired` — a field of the JSON response body, rendered by the
  registry's response serializer from the request it accepted. It is not re-read
  from the registry's storage layer before the response is sent.

Separately, a fleet inventory scanner walks every running instance and every
launch template in the account and writes one signed scan record:

- `InventoryRecorded` — a scan record exists for this image.
- `InventoryVerified` — the service checked the scan record's signature against
  the scanner's key and reports the result here.
- `InventoryCount` — the number of live references to this image that the scan
  counted. When no scan ran, nothing wrote this field.

The two subsystems do not talk to each other, and the evidence loader does not
reconcile them before calling you.

## What the helper is for

`RetirementConfirmed` gates the final step of retirement: on `true` the service
purges the image's backing snapshots and deletes the audit trail that would
otherwise let an operator reconstruct what the image contained. That step cannot
be undone, and any instance still referencing the image can no longer be
diagnosed afterwards.

## Task

Implement `RetirementConfirmed`.
