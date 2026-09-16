package challenge

import (
	"encoding/json"
	"errors"
)

type Grant struct {
	Subject string `json:"subject"`
	Allow   bool   `json:"allow"`
}

func ParseGrant(data []byte) (Grant, error) {
	var raw struct {
		Subject *string `json:"subject"`
		Allow   *bool   `json:"allow"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Grant{}, err
	}
	if raw.Subject == nil || raw.Allow == nil || *raw.Subject == "" {
		return Grant{}, errors.New("missing required member")
	}
	return Grant{Subject: *raw.Subject, Allow: *raw.Allow}, nil
}
