package model

type Mode string

const (
	ModeNosync      Mode = "nosync"
	ModeReplication Mode = "repl"
	ModeCross       Mode = "cross"
)

type ResourceConfig struct {
	Id   string `json:"id"`
	Mode Mode   `json:"mode"`
}
