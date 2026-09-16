package challenge

type ExecutionEvidence struct{ClaimedRun []string;ClaimedPass []string;TrustedObservedRun []string;TrustedObservedPass []string}
func ExecutionProven(required []string,e ExecutionEvidence)bool{m:=map[string]bool{};for _,x:=range e.ClaimedPass{m[x]=true};for _,x:=range required{if !m[x]{return false}};return true}
