package challenge

type Attestation struct {
	SignatureValid bool
	Scope          string
	SubjectDigest  string
}

func AttestationSatisfies(requiredScope string, a Attestation) bool {
	return a.SignatureValid
}
