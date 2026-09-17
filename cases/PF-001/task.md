# PF-001 — duplicate JSON members

Repair `ParseGrant` so valid grants preserve the existing required-field checks (`subject` and `allow` are both present, and `subject` is non-empty), while any exact duplicate member is rejected instead of being resolved by last-write-wins parsing.
