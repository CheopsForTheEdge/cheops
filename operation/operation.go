package operation

import (
	"cheops.com/database"
	"cheops.com/endpoint"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
)

type Operation struct {
	Operation  			string    	`json:"Operation"`
	Sites				[]string 	`json:"Sites"`
	Platform			string      `json:"Platform"`
	Resource   			string  	`json:"Resource"`
	PlatformOperation	string		`json:"PlatformOperation"`
	ExtraArgs			[]string	`json:"ExtraArgs"`
	Request		        string      `json:"Request"`
}

type ExecutionResp struct {
	Site 			string 			 `json:"Site"`
	Request  		string 			 `json:"Request"`
	Response 		http.Response	 `json:"Response"`
}

// Collection name variable
var colname = "operation"

func CreateOperation(operation string,
	sites []string, platform string,
	service string, resource string,
	platformOperation string,
	extraArgs []string, request string) string {
	op := Operation{Operation: operation, Sites: sites,
		Platform: platform, Resource: resource,
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

func ExecuteOperationAPI(w http.ResponseWriter,
						r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal([]byte(reqBody), &op)
	database.CreateResource(colname, op)

	// addresses := SearchEndpoints(op)
	// curl avec search endpoints
	var resps []ExecutionResp
	for _, site := range op.Sites{
		if op.Operation == "&" {
			CreateLeaderFromOperation(op)
		}
		add := endpoint.GetAddress(site)
		exec_add := "http://" + add + ":8080" + "/operation/localrequest"
		resp, err := http.Post(exec_add, "application/json", r.Body)
		if err != nil {
			fmt.Printf("Error in executing command %s \n", exec_add)
			log.Fatal(err)
		}
		execResp := ExecutionResp{"site", "op.Request", *resp}
		resps = append(resps, execResp)
		replication_add := "http://" + add + ":8080" + "replication"
		resp, _ = http.Post(replication_add, "application/json", r.Body)
		execResp = ExecutionResp{"site", "createReplicant", *resp}
		resps = append(resps, execResp)
	}
	// return resps
	json.NewEncoder(w).Encode(resps)
}

func ExecRequestLocally(operation Operation) {
	command := operation.Request
	cmd := exec.Command(command)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Can't exec command %s \n", command)
		log.Fatal(err)
	}
}

func ExecRequestLocallyAPI(w http.ResponseWriter, r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal([]byte(reqBody), &op)
	ExecRequestLocally(op)
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
