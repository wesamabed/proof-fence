package challenge

type Fact struct{Known bool;Value bool}
type Evidence struct{ProducerPresent bool;Authenticated bool;DerivedSafe bool}
func NetworkSafety(e Evidence)Fact{return Fact{Known:true,Value:e.DerivedSafe}}
