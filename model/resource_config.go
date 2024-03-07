package model

type Mode string

const (
	ModeReplication Mode = "repl"
	ModeCross       Mode = "cross"
)

type OperationType string

const (
	// Idempotent and Commutative (Type 1)
	OperationTypeCommutativeIdempotent OperationType = "A"

	// Commutative only (Type 2)
	OperationTypeCommutative OperationType = "B"

	// Idempotent only (Type 3)
	OperationTypeIdempotent OperationType = "C"

	// Not commutative, not idempotent (Type 4)
	OperationTypeNothing OperationType = "D"
)

type ResourceConfig struct {
}
