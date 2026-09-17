# PF-002 — exact JSON vocabulary

Repair `ParseBinding` so only exact lower-case members `run_id` and `authorized` are admitted. Reject case aliases such as `Run_ID`, including when an exact key is also present.

`run_id` and `authorized` are the parser's whole vocabulary. A member outside it
is rejected with an error rather than silently dropped: this parser refuses any
document it does not fully understand, so admitting a document is a positive
statement that every member in it was recognised.
