package model

type Mode string

const (
	ModeReplication Mode = "repl"
	ModeCross       Mode = "cross"
)

type ResourceConfig struct {
}
