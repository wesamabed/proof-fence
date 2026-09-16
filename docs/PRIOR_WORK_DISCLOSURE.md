# Prior-work disclosure

ProofFence was motivated by private defensive engineering work on **Arbiter**, a security-oriented Go/PostgreSQL cloud control-plane project maintained by Wesam Abed with extensive AI-assisted implementation and review.

The private work produced recurring examples of evidence-provenance, parsing, outcome-coherence, cleanup, and CI-verification failures. ProofFence abstracts those *general classes* into new synthetic examples.

Important boundaries:
- ProofFence is a separate public artifact, not a publication of Arbiter.
- No Arbiter source code is copied into the v0.1 cases.
- No credentials, owner signing material, live AWS identifiers, or private infrastructure are included.
- Private AI review reports are not included.
- Internal Arbiter findings are not represented as CVEs, third-party OSS vulnerabilities, or external audits.
- ProofFence v0.1 does not claim novelty relative to the academic or industry literature; a prior-art review is required before such a claim.
