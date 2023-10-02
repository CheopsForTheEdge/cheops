package env

import (
	"log"
	"os"
)

var Myfqdn string

func init() {

	fqdn, ok := os.LookupEnv("MYFQDN")
	if !ok {
		log.Fatal("My FQDN must be given with the MYFQDN environment variable !")
	}
	Myfqdn = fqdn
}
