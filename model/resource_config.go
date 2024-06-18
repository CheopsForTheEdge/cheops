package model

type Mode string

const (
	ModeReplication Mode = "repl"
	ModeCross       Mode = "cross"
)

type ResourceConfig struct {
	RelationshipMatrix []Relationship
}

type Relationship struct {
	Before OperationType
	After  OperationType
	Result []int
}
