// Package utils handles every utility functions, like configurations,
// heartbeat, database.
package utils

import (
	"github.com/segmentio/ksuid"
)


func CreateMetaId() string {
	id := ksuid.New()
	cheopsID := "CHEOPS_" + id.String()
	return cheopsID
}

/*
// Heartbeat sends a heartbeat to the given site and updates the latency to it
func Heartbeat(site endpoint.Site) {
	host := site.Address
	port := Conf.Application.HeartbeatPort
	timeout := time.Duration(1 * time.Second)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host + ":" + port, timeout)
	if err != nil {
		fmt.Printf("%s %s %s\n", host, "not responding", err.Error())
		log.Fatal(err)
	} else {
		fmt.Printf("%s %s %s\n", host, "responding on port:", port)
	}
	latency := time.Since(start)
	var s endpoint.Site
	_, id := SearchResource("sites", "SiteName",
		site.SiteName, &s)
	update := map[string]interface{}{"Latency": latency}
	UpdateResource("sites", id, update)
	conn.Close()
}



func SendHeartbeats() {
	var interf []interface{}
	interf = GetAll(interf, "sites")
	if interf == nil {
		log.Fatal()
		fmt.Println("Document cannot be read." )
	}
	for _, site := range interf {
		// TODO check if it could be possible to send the key also
		Heartbeat(site.(endpoint.Site))
	}
}
*/
