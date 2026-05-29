package patch

type Options struct {
	Write bool
}

type Result struct {
	Changed bool
	Diff    string
}

func Apply(Options) (Result, error) {
	return Result{}, nil
}
