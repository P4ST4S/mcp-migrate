package live

type STDIOTrace struct {
	Command              string
	AllowMutatingProbes  bool
	SentLegacyInitialize bool
	StderrBytes          int
	StderrTruncated      bool
	Observations         []STDIOObservation
}

type STDIOObservation struct {
	Probe        string
	RPCMethod    string
	ReadOnly     bool
	Mutating     bool
	MetaIncluded bool
	HasRPCError  bool
	RPCErrorCode int
	HasResult    bool
	Result       map[string]any
	Timeout      bool
	ProcessError bool
	ParseError   bool
}

func (o STDIOObservation) Accepted() bool {
	return !o.Timeout && !o.ProcessError && !o.ParseError && !o.HasRPCError && o.HasResult
}
