package operation

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"cheops.com/database"
	"cheops.com/endpoint"
)

type Operation struct {
	Operation  			string    	`json:"Operation"`
	Sites				[]string 	`json:"Sites"`
	Platform			string      `json:"Platform"`
	Service 			string    	`json:"Service"`
	Resource   			string  	`json:"Resource"`
	PlatformOperation	string		`json:"PlatformOperation"`
	ExtraArgs			[]string	`json:"ExtraArgs"`
	Request		        string      `json:"Request"`
}

// Collection name variable
var colname = "operation"

func CreateOperation(operation string,
	sites []string, platform string,
	service string, resource string,
	platformOperation string,
	extraArgs []string, request string) string {
	op := Operation{Operation: operation, Sites: sites,
		Platform: platform, Service: service, Resource: resource,
		PlatformOperation: platformOperation, ExtraArgs: extraArgs,
		Request: request}
	return database.CreateResource(colname, op)
}


func CreateOperationAPI(w http.ResponseWriter, r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal([]byte(reqBody), &op)
	key := database.CreateResource(colname, op)
	json.NewEncoder(w).Encode(key)
}

func ExecuteOperationAPI(w http.ResponseWriter, r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal([]byte(reqBody), &op)
	key := database.CreateResource(colname, op)
	json.NewEncoder(w).Encode(key)
	addresses := SearchEndpoints(op)
	// curl avec search endpoints
}


func SearchEndpoints(op Operation) []string {
	var addresses []string
	for _, site := range op.Sites{
		address := endpoint.GetAddress(site)
		addresses = append(addresses, address)
	}
	return addresses
}


func SendRequestToBroker(op Operation) {
	// call to Broker API with address and the op jsonified
}
