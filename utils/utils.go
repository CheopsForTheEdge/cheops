package utils

import (
	"encoding/json"
	"fmt"
	"github.com/segmentio/ksuid"
	"os"
)

type Configuration struct {
	Site    string `json:"Site"`
	Address string `json:"Address"`

}

func CreateMetaId() string {
	id := ksuid.New()
	return id.String()
}

func GetConfig() (conf Configuration) {
	file, _ := os.Open("conf.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("error:", err)
	}
	return conf
}