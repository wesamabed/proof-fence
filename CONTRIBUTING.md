# Contributing

Contributions should preserve the benchmark's defensive scope and evidence hierarchy.

A new case should:
1. state one security invariant;
2. use synthetic identities and fixtures;
3. include a starter implementation that violates the invariant;
4. include a grader that catches the violation;
5. include a reference implementation for mechanical self-test;
6. avoid private source code or exploit-ready details tied to real infrastructure;
7. document what the case does and does not prove.

Case IDs are monotonic (`PF-011`, `PF-012`, ...). Do not repurpose existing IDs.
