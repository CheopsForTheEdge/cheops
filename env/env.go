package env

import (
	"log"
	"os"
)

var Myip string
var Myfqdn string

func init() {
	ip, ok := os.LookupEnv("MYIP")
	if !ok {
		log.Fatal("My IP must be given with the MYIP environment variable !")
	}
	Myip = ip

	fqdn, ok := os.LookupEnv("MYFQDN")
	if !ok {
		log.Fatal("My FQDN must be given with the MYFQDN environment variable !")
	}
	Myfqdn = fqdn
}
