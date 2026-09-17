package challenge

type ExecutionEvidence struct {
	SelfReportRan    []string
	SelfReportPassed []string
	SupervisorRan    []string
	SupervisorPassed []string
}

func ExecutionEstablished(required []string, e ExecutionEvidence) bool {
	passed := map[string]bool{}
	for _, name := range e.SelfReportPassed {
		passed[name] = true
	}
	for _, name := range required {
		if !passed[name] {
			return false
		}
	}
	return true
}
