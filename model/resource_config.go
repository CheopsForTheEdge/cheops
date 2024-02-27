package model

import (
	"encoding/json"
	"log"
)

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

func ValidateConfig(b []byte) bool {
	var config ResourceConfig
	err := json.Unmarshal(b, &config)
	if err != nil {
		log.Printf("Invalid config file: %v", err)
		return false
	}

	if config.OperationsType == "" {
		return false
	}

	return true
}
