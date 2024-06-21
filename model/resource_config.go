package model

type Mode string

const (
	ModeReplication Mode = "repl"
	ModeCross       Mode = "cross"
)

type ResourceConfig struct {
	RelationshipMatrix []Relationship
}

type RelationshipType string

const (
	TakeOne              RelationshipType = "take-one"
	TakeBothAnyOrder     RelationshipType = "take-both-any-order"
	TakeBothKeepOrder    RelationshipType = "take-both-keep-order"
	TakeBothReverseOrder RelationshipType = "take-both-reverse-order"
)

type Relationship struct {
	Before OperationType
	After  OperationType
	Result RelationshipType
}
