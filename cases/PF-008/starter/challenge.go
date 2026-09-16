package challenge

type CleanupEvidence struct{DeleteOutcome Outcome;ErrorCode string;ResponseDeleted bool;ResidueObserved bool;ResidueAuthenticated bool;ResidueEmpty bool}
type Outcome string
const(Succeeded Outcome="SUCCEEDED";Denied Outcome="DENIED";Failed Outcome="FAILED")
func CleanupComplete(e CleanupEvidence)bool{return e.ResponseDeleted&&e.ResidueEmpty}
