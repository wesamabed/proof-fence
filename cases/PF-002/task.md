# PF-002 — exact JSON vocabulary

Repair `ParseBinding` so only exact lower-case members `run_id` and `authorized` are admitted. Reject case aliases such as `Run_ID`, including when an exact key is also present.
