package api

import (
	"log"
	"net/http"
	"os"
)

var myip string
var myfqdn string

func init() {
	ip, ok := os.LookupEnv("MYIP")
	if !ok {
		log.Fatal("My IP must be given with the MYIP environment variable !")
	}
	myip = ip

	fqdn, ok := os.LookupEnv("MYFQDN")
	if !ok {
		log.Fatal("My FQDN must be given with the MYFQDN environment variable !")
	}
	myfqdn = fqdn

	m, ok := os.LookupEnv("MODE")
	if !ok {
		log.Fatal("My FQDN must be given with the MYFQDN environment variable !")
	}
	switch m {
	case "raft":
		mode = raftMode
	case "crdt":
		mode = crdtMode
	default:
		log.Fatalf("Invalid MODE, want 'raft' or 'crdt', got [%v]\n", m)
	}
}

type Payload struct {
	RequestId string
	Method    string
	Header    http.Header
	Path      string
	Body      []byte
}
