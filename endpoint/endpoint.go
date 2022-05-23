package endpoint

import (
	"cheops.com/utils"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)


// Endpoints are for services on a site
type Endpoint struct {
	Service 	string `json:"Service"`
	Address 	string `json:"Address"`
}

// Collection name variable
var colname = "endpoints"

// CreateEndpoint Constructor
func CreateEndpoint(service string, address string) string {
	end := Endpoint{Service: service, Address: address}
	return utils.CreateResource(colname, end)
}

func CreateEndpointAPI(w http.ResponseWriter, r *http.Request) {
	service := mux.Vars(r)["Service"]
	add := mux.Vars(r)["Address"]
	key := CreateEndpoint(service, add)
	json.NewEncoder(w).Encode(key)
}

func GetEndpointAddress(service string) string {
	query := "FOR end IN endpoint FILTER end.Service == @name RETURN end"
	bindvars := map[string]interface{}{ "name": service }
	result := Endpoint{}
	utils.ExecuteQuery(query, bindvars, &result)
	if result.Address == "" {
		err := fmt.Sprintf("Address %s not found.\n", service)
		fmt.Print(err)
		log.Fatal(err)
	}
	return result.Address
}


func GetEndpointAddressAPI(w http.ResponseWriter, r *http.Request) {
	service := mux.Vars(r)["Service"]
	add := GetEndpointAddress(service)
	if add != "" {
		json.NewEncoder(w).Encode(add)
		return
	}
	w.WriteHeader(404)
}

// Contact an endpoint with a GET
func ContactEndpoint(service string) *http.Response  {
	address := GetEndpointAddress(service)
	response,_ := http.Get(address)
	return response
}
