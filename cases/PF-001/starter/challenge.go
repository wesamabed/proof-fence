package challenge

import "encoding/json"

type Grant struct { Subject string `json:"subject"`; Allow bool `json:"allow"` }

func ParseGrant(data []byte) (Grant, error) {
    var g Grant
    err := json.Unmarshal(data, &g)
    return g, err
}
