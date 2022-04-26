package operation

import (
	"cheops.com/database"
	"cheops.com/endpoint"
	"cheops.com/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type Operation struct {
	Operation         string   `json:"Operation"`
	Sites             []string `json:"Sites"`
	Platform          string   `json:"Platform"`
	Resource          string   `json:"Resource"`
	Instance          string   `json:"Instance"`
	PlatformOperation string   `json:"PlatformOperation"`
	ExtraArgs         []string `json:"ExtraArgs"`
	Request           string   `json:"Request"`
}

type ExecutionResp struct {
	Site     string        `json:"Site"`
	Request  string        `json:"Request"`
	Response http.Response `json:"Response"`
}

// Collection name variable
var colname = "operations"

var config = utils.GetConfig()

func CreateOperation(operation string,
	sites []string, platform string,
	service string, resource string,
	instance string,
	platformOperation string,
	extraArgs []string, request string) string {
	op := Operation{Operation: operation, Sites: sites,
		Platform: platform, Resource: resource, Instance: instance,
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
	// create a table for responses
	var resps []ExecutionResp
	// Executing operations on each sites, might need threads to do it in parallel
	for _, site := range op.Sites {
		add := endpoint.GetSiteAddress(site)
		// using the ExecRequestLocally on each involved site
		execAdd := "http://" + add + ":8080" + "/operation/localrequest"
		// for post, we need a reader, so we need the operation marshalled
		operation, _ := json.Marshal(op)
		opReader := strings.NewReader(string(operation))
		// execute the actual request
		resp, err := http.Post(execAdd, "application/json",
			opReader)
		if err != nil {
			fmt.Printf("Error in executing command %s \n", execAdd)
			log.Fatal(err)
		}
		// create the response
		execResp := ExecutionResp{"site", "op.Request", *resp}
		resps = append(resps, execResp)
		// depending on the operation, we have to do stuff (e.g.
		// create the replicants)
		if op.Operation == "&" {
			//TODO: maybe create this in replication so we only have to call it
			if op.PlatformOperation == "create" {
				replicationAdd := "http://" + add + ":8080" + "/replication"
				resp, _ = http.Post(replicationAdd, "application/json", opReader)
				if resp != nil {
					execResp = ExecutionResp{"site", "createReplicant", *resp}
					resps = append(resps, execResp)
				}
			}
			if op.PlatformOperation == "update" {
				//TODO: call the API instead (through the broker)
				if CheckIfReplicant(op.Instance) {
					// Check if leader
				}
			}
			if op.PlatformOperation == "delete" {
				//TODO: call the API instead (through the broker)
				if CheckIfReplicant(op.Instance) {

				}
			}
		}
	}
	// return resps
	json.NewEncoder(w).Encode(resps)
}

func ExecRequestLocally(operation Operation) (out string) {
	// slice the request
	f := strings.Fields(operation.Request)
	// the program called is the first word in the slice
	command := f[0]
	// the args are the rest, as string
	arg := f[1:]
	// exec the entire request
	cmd := exec.Command(command, arg...)
	stdout, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Can't exec command %s: %s \n", command, stdout)
		fmt.Printf("Stdout %s \n", stdout)
		return string(stdout)
	}
	return string(stdout)
}

func ExecRequestLocallyAPI(w http.ResponseWriter, r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal([]byte(reqBody), &op)
	out := ExecRequestLocally(op)
	w.Write([]byte(out))
}

func SearchSites(op Operation) []string {
	var addresses []string
	for _, site := range op.Sites {
		address := endpoint.GetSiteAddress(site)
		addresses = append(addresses, address)
	}
	return addresses
}

func SendRequestToBroker(op Operation) {
	// call to Broker API with address and the op jsonified
}
