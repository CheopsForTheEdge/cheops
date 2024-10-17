package model

type Mode string

const (
	ModeReplication Mode = "repl"
	ModeCross       Mode = "cross"
)

type ResourceConfig struct {
	ResolutionMatrix []Resolution
}

func (c ResourceConfig) IsEmpty() bool {
	return len(c.ResolutionMatrix) == 0
}

type ResolutionType string

const (
	TakeOne              ResolutionType = "take-one"
	TakeBothAnyOrder     ResolutionType = "take-both-any-order"
	TakeBothKeepOrder    ResolutionType = "take-both-keep-order"
	TakeBothReverseOrder ResolutionType = "take-both-reverse-order"
)

type Resolution struct {
	Before OperationType
	After  OperationType
	Result ResolutionType
}
