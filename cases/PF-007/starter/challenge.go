package challenge

type Reachability struct{CallerSaysIsolated bool;ObservationPresent bool;ObservationAuthenticated bool;ObservedIsolated bool}
func IsolationProven(r Reachability)bool{if r.CallerSaysIsolated{return true};return r.ObservedIsolated}
