package utils

import (
	"cheops.com/endpoint"
	"encoding/json"
	"fmt"
	"github.com/segmentio/ksuid"
	"net"
	"os"
	"time"
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
func Heartbeat(site endpoint.Site) {
	host := site.Address
	port := Conf.Application.HeartbeatPort
	timeout := time.Duration(1 * time.Second)
	_, err := net.DialTimeout("tcp", host + ":" + port, timeout)
	if err != nil {
		fmt.Printf("%s %s %s\n", host, "not responding", err.Error())
	} else {
		fmt.Printf("%s %s %s\n", host, "responding on port:", port)
	}
}

func SendHeartbeats() (sitesnames []string){
	var sites []string

	return sites
}