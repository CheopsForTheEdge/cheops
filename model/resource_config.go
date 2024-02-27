package model

type Mode string

const (
	ModeReplication Mode = "repl"
	ModeCross       Mode = "cross"
)

type OT string

const (
	// Idempotent and Commutative (Type 1)
	OperationsTypeCommutativeIdempotent OT = "A"

	// Commutative only (Type 2)
	OperationsTypeCommutative OT = "B"

	// Idempotent only (Type 3)
	OperationsTypeIdempotent OT = "C"

	// Not commutative, not idempotent (Type 4)
	OperationsTypeNothing OT = "D"
)

type ResourceConfig struct {
	Id             string `json:"id"`
	OperationsType OT
}
