package utils

import (
	"cheops.com/endpoint"
	"fmt"
	"github.com/segmentio/ksuid"
	"net"
	"time"
)


func CreateMetaId() string {
	id := ksuid.New()
	cheopsID := "CHEOPS_" + id.String()
	return cheopsID
}


// TODO maybe use httpstat https://pkg.go.dev/github.com/tcnksm/go-httpstat
func Heartbeat(site endpoint.Site) {
	host := site.Address
	port := Conf.Application.HeartbeatPort
	timeout := time.Duration(1 * time.Second)
	_, err := net.DialTimeout("tcp", host + ":" + port, timeout)
	// TODO close connection
	if err != nil {
		fmt.Printf("%s %s %s\n", host, "not responding", err.Error())
	} else {
		fmt.Printf("%s %s %s\n", host, "responding on port:", port)
	}
}

func SendHeartbeats() (){
/*	var interf []endpoint.Site
	sites := GetAll(interf, "sites")
	for _, site := range sites {

	}*/
}