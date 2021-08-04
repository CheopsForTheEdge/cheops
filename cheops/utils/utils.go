package Utils

import (
	"github.com/segmentio/ksuid"
)

func CreateMetaId() string {
	id := ksuid.New()
	return id.String()
}
