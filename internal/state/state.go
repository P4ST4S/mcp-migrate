package state

type Signal struct {
	ID      string
	Message string
}

func DetectHiddenState() []Signal {
	return nil
}
