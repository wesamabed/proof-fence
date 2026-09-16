# PF-004 — removal response envelope

`challenge.go` is a client-side helper for a record-storage service.

## The API

`DELETE /records/{id}` returns one `RemovalResponse` envelope. Its three parts
are populated by three different components of the service and returned as-is by
the transport:

- `Result` — the request router's classification of the invocation. The router
  sets exactly one of `SUCCEEDED`, `DENIED`, or `FAILED`. Any other value comes
  from a service version this client release does not model.
- `FaultCode` — empty when the service's fault recorder logged no fault while
  handling the request; otherwise the recorded fault identifier. The fault
  recorder writes this field from its own log, independently of how the router
  classified `Result`.
- `PayloadRemoved` — a field of the JSON response body. The body is rendered by
  the response serializer from the request the service accepted. It is not
  re-read from the storage layer before the response is sent.

The client issues one call and gets one envelope. There is no follow-up read.

## What the helper is for

`RemovalConfirmed` is called by the retention pipeline. A `true` result is
treated as a durable record that the object is gone: the pipeline writes a
deletion receipt, drops the id from the re-scan queue, and never revisits it.
A `false` result leaves the id queued for another pass.

## Task

Implement `RemovalConfirmed` so that it reports `true` exactly when this single
envelope establishes that the record is gone.
