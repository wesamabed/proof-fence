# PF-010 — execution evidence

`challenge.go` is part of a CI gate that decides whether a pull request's
required security tests actually ran.

## The API

Two independent sources describe the same test run.

The job under test writes a summary file just before its process exits. The gate
parses that file into `SelfReportRan` and `SelfReportPassed`: the test names the
job states it started, and the names it states finished successfully. The file is
produced inside the job, by the job's own code, from whatever that code chose to
write — including code the pull request under review changed.

Separately, the gate supervises the test process from outside it. It consumes the
process's structured test-event stream as events are emitted and records
`SupervisorRan` (names it observed a test-start event for) and `SupervisorPassed`
(names it observed a success event for). The supervisor appends only events it
observed itself, as they arrived. A test that starts emits a start event, and a
test that then succeeds emits a success event, so a complete stream names every
passing test in both lists. The two supervisor lists accumulate independently, so
a truncated or dropped stream can leave them inconsistent with each other: a name
in one list and not the other is the signature of an incomplete stream, not of a
test that behaved unusually.

Either source may name tests that are not in `required`: repositories run more
tests than any one gate requires. Test names are compared literally, and
`required` is supplied non-empty by the gate's policy loader.

## What the helper is for

`ExecutionEstablished(required, e)` gates merge. A `true` result merges the pull
request with no human looking at the security tests; a `false` result blocks the
merge and requests a maintainer review.

## Task

Implement `ExecutionEstablished`.
