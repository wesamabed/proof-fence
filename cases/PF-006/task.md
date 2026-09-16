# PF-006 — foreign valid context

Repair `AuthorizedForRun`. A context may authorize only when it is verified, has a non-empty subject digest, and its `RunID` exactly matches the required run.
