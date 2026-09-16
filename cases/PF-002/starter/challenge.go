package challenge

import "encoding/json"

type Binding struct { RunID string `json:"run_id"`; Authorized bool `json:"authorized"` }
func ParseBinding(data []byte) (Binding,error) { var b Binding; err:=json.Unmarshal(data,&b); return b,err }
