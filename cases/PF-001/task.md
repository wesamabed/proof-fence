# PF-001 — duplicate JSON members

Repair `ParseGrant` so a valid grant parses normally, but an exact duplicate member such as two `subject` keys is rejected. Do not weaken required-field validation.
