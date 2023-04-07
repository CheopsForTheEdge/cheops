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



func Heartbeat(site endpoint.Site) {
	host := site.Address
	port := Conf.Application.HeartbeatPort
	timeout := time.Duration(1 * time.Second)
	conn, err := net.DialTimeout("tcp", host + ":" + port, timeout)
	// TODO add latency count
	if err != nil {
		fmt.Printf("%s %s %s\n", host, "not responding", err.Error())
	} else {
		fmt.Printf("%s %s %s\n", host, "responding on port:", port)
	}
	conn.Close()
}


// TODO add errors
func SendHeartbeats() {
	var interf []interface{}
	interf = GetAll(interf, "sites")
	for _, site := range interf {
		Heartbeat(site.(endpoint.Site))
	}
}