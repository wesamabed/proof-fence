# PF-009 — required-test disappearance

Repair `ValidateRequiredTests`. Return an error if any required test is absent from the discovered test list or if the required manifest itself contains duplicates. Extra discovered tests are allowed.
