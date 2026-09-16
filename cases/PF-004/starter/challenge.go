package challenge

type Outcome string
const(Succeeded Outcome="SUCCEEDED";Denied Outcome="DENIED";Failed Outcome="FAILED")
type DeleteEvidence struct{Outcome Outcome;ErrorCode string;ResponseDeleted bool}
func DeletionComplete(e DeleteEvidence)bool{return e.ResponseDeleted}
