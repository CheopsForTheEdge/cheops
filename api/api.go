package api

import (
	"log"
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

type Request struct {
	Method string
	Path   string
	Body   string
}
