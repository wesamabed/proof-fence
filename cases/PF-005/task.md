# PF-005 — analyzer run records

`challenge.go` is part of a network-exposure console.

## The API

A reachability analyzer runs on a schedule. Each completed run writes one signed
run record for the subject it analyzed. The console loads the latest record for a
subject and hands your function a `PathAnalysis` value:

- `Recorded` — a run record exists for this subject.
- `SignatureValid` — the console checked the record's detached signature against
  the analyzer fleet's published key and reports the result here. A record whose
  signature does not verify is not attributable to the analyzer; its bytes may
  have come from anywhere.
- `ReportedNoPath` — a field stored inside the record, set by the analyzer to the
  conclusion it reached: whether it found no network path from the public edge to
  the subject.

The three fields are filled in by three separate steps of the console's evidence
loader and are handed to your function without being cross-checked.

## What the helper is for

`PathFinding` returns a `Finding` with two fields, `Determined` and `Value`. The
console renders exactly three states to the operator, and it reads `Value` only
when `Determined` is true:

- `Determined: true, Value: true` — displayed as "no public path". The subject is
  removed from the exposure worklist and is not looked at again this cycle.
- `Determined: true, Value: false` — displayed as "public path exists". The
  subject is escalated to the on-call queue as a live exposure.
- `Determined: false` — displayed as "not analyzed". The console schedules an
  analyzer run and displays nothing at all about this subject's exposure.

## Task

Implement `PathFinding` so that every input produces the console state that the
available evidence supports.
