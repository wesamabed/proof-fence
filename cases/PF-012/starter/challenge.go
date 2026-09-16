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

type ChangeRequest struct {
	Subject    string
	Generation int
}

type Observation struct {
	Present    bool
	Verified   bool
	Subject    string
	Generation int
	Compliant  bool
}

func AuthorityDecision(req ChangeRequest, obs Observation) Decision {
	if obs.Verified && obs.Compliant {
		return Retain
	}
	return Revoke
}
