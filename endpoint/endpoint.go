package endpoint

import (
	"net/http"
	"cheops/database"
)


type Endpoint struct {
	Service string `json:"Service"`
	Address string `json:"Address"`
}

// Collection name variable
var colname = "endpoint"

// Constructor
func CreateEndpoint(service string, address string) string {

	end := Endpoint{Service: service, Address: address}
	return database.CreateResource(colname, end)
}

// Contact an endpoint with a GET
func ContactEndpoint(endpoint Endpoint) *http.Response  {
	response,_ := http.Get(endpoint.Address)
	return response
}
