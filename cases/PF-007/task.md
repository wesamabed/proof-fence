# PF-007 — containment inputs

`challenge.go` runs inside a tenant-isolation service.

## The API

`POST /containment/check` accepts a request body in which the caller states its
own view of the subject's network containment. The service copies that field
verbatim into `RequestField`. Any tenant-side caller holding a tenant API token
can send this request and can put either value in that field.

Separately, the service's probe fleet runs network probes against subjects from
outside the tenant boundary. A completed probe stores one signed probe record:

- `ProbeRecorded` — a probe record exists for this subject.
- `ProbeVerified` — the service checked the probe record's signature against the
  probe fleet's key and reports the result here.
- `ProbeResult` — the containment value the probe measured and wrote into the
  record.

`RequestField` and the three probe fields are assembled by separate steps of the
service's evidence loader and are handed to your function without being
cross-checked against one another.

## What the helper is for

`ContainmentEstablished` gates a control-plane operation that is safe to run only
against a contained subject: on `true` the operation starts immediately with no
further checks. On `false` the request is answered with a retry-later status and
a probe is scheduled.

## Task

Implement `ContainmentEstablished`.
