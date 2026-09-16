# PF-003 — trailing document

Repair `ParseDecision` so one valid JSON object plus whitespace is accepted, but a second JSON value or trailing non-whitespace content is rejected.
