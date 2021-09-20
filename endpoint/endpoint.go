package endpoint

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"cheops.com/database"
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

func GetAddress(site string) string {
	query := "FOR end IN endpoint FILTER end.Service == @name RETURN end"
	bindvars := map[string]interface{}{ "name": site }
	result := Endpoint{}
	database.ExecuteQuery(query, bindvars, &result)
	return result.Address
}


func GetAddressAPI(w http.ResponseWriter, r *http.Request) {
	site := mux.Vars(r)["Site"]
	add := GetAddress(site)
	if add != "" {
		json.NewEncoder(w).Encode(add)
		return
	}
	w.WriteHeader(404)
}

// Contact an endpoint with a GET
func ContactEndpoint(site string) *http.Response  {
	address := GetAddress(site)
	response,_ := http.Get(address)
	return response
}
