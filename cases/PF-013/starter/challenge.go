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

type Intent string

const (
	Install Intent = "INSTALL"
	Remove  Intent = "REMOVE"
)

type ApplyStatus string

const (
	Applied  ApplyStatus = "APPLIED"
	Rejected ApplyStatus = "REJECTED"
	Errored  ApplyStatus = "ERROR"
)

type SlotState string

const (
	KeyPresent SlotState = "KEY_PRESENT"
	KeyAbsent  SlotState = "KEY_ABSENT"
	Unreadable SlotState = "UNREADABLE"
)

type Request struct {
	Intent Intent
}

type ApplyReport struct {
	Returned bool
	Status   ApplyStatus
}

type Readback struct {
	Present  bool
	Verified bool
	State    SlotState
}

func RotationDecision(req Request, report ApplyReport, rb Readback) Decision {
	if report.Status == Applied {
		if req.Intent == Install {
			return Grant
		}
		return Revoke
	}
	return Retain
}
