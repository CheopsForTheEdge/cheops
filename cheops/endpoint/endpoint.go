package endpoint

import "net/http"

type Endpoint struct {
	service		string  `json:"service"`
	address 	string  `json:"address"`
}


func NewEndpoint(service string, address string) *Endpoint {
	return &Endpoint{service: service, address: address}
}


func contactEndpoint(endpoint Endpoint) *http.Response  {
	response,_ := http.Get(endpoint.address)
	return response
}

