package challenge
import("bytes";"encoding/json")
type Decision struct{Permit bool `json:"permit"`}
func ParseDecision(data []byte)(Decision,error){var v Decision; d:=json.NewDecoder(bytes.NewReader(data)); e:=d.Decode(&v); return v,e}
