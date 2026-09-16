# PF-005 — missing producer

Repair `NetworkSafety`. `Safe` is known only when the required producer is present and authenticated. If either premise is missing, return an unknown fact rather than a positive or negative conclusion.
