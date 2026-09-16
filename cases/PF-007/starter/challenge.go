package challenge

type ContainmentInputs struct {
	RequestField  bool
	ProbeRecorded bool
	ProbeVerified bool
	ProbeResult   bool
}

func ContainmentEstablished(in ContainmentInputs) bool {
	if in.RequestField {
		return true
	}
	return in.ProbeResult
}
