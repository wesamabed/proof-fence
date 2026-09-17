# PF-003 — trailing document

Repair `ParseDecision` so one valid JSON object plus whitespace is accepted, but a second JSON value or trailing non-whitespace content is rejected.

The one value must itself be a JSON object. Any other JSON value is rejected,
including `null`, an array, a number, a string, and a boolean. Note that Go
decodes some of these into a struct without reporting an error.
