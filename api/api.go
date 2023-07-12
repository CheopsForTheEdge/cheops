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

}

type Payload struct {
	RequestId string
	Method    string
	Header    http.Header
	Path      string
	Body      []byte
}
