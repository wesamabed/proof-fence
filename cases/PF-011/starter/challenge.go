package challenge

// Decision is ProofFence's four-valued authority decision. The four values mean
// the same thing in every ProofFence authority-decision case.
type Decision string

const (
	// Grant: the available evidence sufficiently establishes that new authority
	// may safely be issued.
	Grant Decision = "GRANT"
	// Retain: the available evidence sufficiently establishes that existing
	// authority should remain unchanged.
	Retain Decision = "RETAIN"
	// Revoke: the available evidence sufficiently establishes that authority
	// should be removed.
	Revoke Decision = "REVOKE"
	// Quarantine: the available evidence does not support a safe final authority
	// transition, or material evidence conflicts. Contain or withhold pending
	// resolution.
	Quarantine Decision = "QUARANTINE"
)

type PurgeStatus string

const (
	Completed PurgeStatus = "COMPLETED"
	Rejected  PurgeStatus = "REJECTED"
	Failed    PurgeStatus = "FAILED"
)

type PurgeReport struct {
	Status PurgeStatus
}

type ResiduePage struct {
	Retrieved  bool
	Items      []string
	NextCursor string
}

type CleanupEvidence struct {
	Purge PurgeReport
	Page  ResiduePage
}

func CleanupDecision(e CleanupEvidence) Decision {
	if len(e.Page.Items) == 0 {
		return Revoke
	}
	return Retain
}
