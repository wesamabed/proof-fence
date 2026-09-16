# PF-006 — foreign valid context

Repair `AuthorizedForRun`. The benchmark supplies a non-empty required run. A context may authorize only when it is verified, has a non-empty subject digest, and its `RunID` exactly matches that required run.
