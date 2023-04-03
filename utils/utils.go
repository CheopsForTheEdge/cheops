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
	cheopsID := "CHEOPS_" + id.String()
	return cheopsID
}

// No longer used
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


// TODO maybe use httpstat https://pkg.go.dev/github.com/tcnksm/go-httpstat
func Heartbeat(SiteName string) {
}

func SendHeartbeats() (sitesnames []string){
	var sites []string

	return sites
}