package challenge

type VerifiedContext struct{Verified bool;RunID string;SubjectDigest string}
func AuthorizedForRun(required string,c VerifiedContext)bool{return c.Verified}
