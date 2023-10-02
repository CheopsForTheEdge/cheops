// Package operation manages the generic operations in Cheops,
// keeping all necessary information and providing different functions to
// manage the different possible operations.
package operation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"cheops.com/utils"
)

// Operation is the generic type to describe operations in Cheops.
// Operation is the operator (collaboration) used, like ',', '&',
// '%'. Sites is the list of sites needed for the operation like '["Site1",
// "Site2"]'. Platform is the application used like 'openstack' or
// 'kubernetes'. Resource is the involved resource, such as 'server',
// 'deployment'. Instance. PlatformOperation is the crud operation involved,
// for example 'create',
// 'delete'. Request is the full request that was made to deduce this
// operation and ExtraArgs if there was something more in the request needed
// for the operation. Redirection is the bool to know whether the operation
// was requested directly on this site or if it was sent from another site.
type Operation struct {
	Operation         string   `json:"Operation"`
	Sites             []string `json:"Sites"`
	Platform          string   `json:"Platform"`
	Resource          string   `json:"Resource"`
	Instance          string   `json:"Instance"`
	PlatformOperation string   `json:"PlatformOperation"`
	ExtraArgs         []string `json:"ExtraArgs"`
	Request           string   `json:"Request"`
	Redirection       bool
}

type ExecutionResp struct {
	Site     string        `json:"Site"`
	Request  string        `json:"Request"`
	Response http.Response `json:"Response"`
}

// Collection name variable
var colname = "operations"

var conf = utils.Conf

func CreateOperation(operation string,
	sites []string, platform string,
	service string, resource string,
	instance string,
	platformOperation string,
	extraArgs []string, request string) string {
	op := Operation{Operation: operation, Sites: sites,
		Platform: platform, Resource: resource, Instance: instance,
		PlatformOperation: platformOperation, ExtraArgs: extraArgs,
		Request: request, Redirection: false}
	return utils.CreateResource(colname, op)

}

func CreateOperationAPI(w http.ResponseWriter, r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal([]byte(reqBody), &op)
	op.Redirection = false
	key := utils.CreateResource(colname, op)
	json.NewEncoder(w).Encode(key)
}

func ExecuteOperationAPI(w http.ResponseWriter, r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal([]byte(reqBody), &op)
	if err != nil {
		fmt.Fprintf(w, "There was an error reading the json: %s\n ",
			err)
		log.Fatal(err)
	}
	utils.CreateResource(colname, op)
	// create a table for responses
	//var resps []ExecutionResp
	// First, check if this is a redirection to know if we need to read sites
	if !(op.Redirection) {
		if op.Operation == "&" {
			ExecuteReplication(op, conf)
		}

	}
}

// return resps
//json.NewEncoder(w).Encode(resps)

// ExecRequestLocally takes an operation and executes the command locally.
// It returns the response or error while executing the command.
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

// ExecRequestLocallyAPI
func ExecRequestLocallyAPI(w http.ResponseWriter, r *http.Request) {
	op := ReadOperationFromRequest(r, w)
	out := ExecRequestLocally(op)
	_, err := w.Write([]byte(out))
	if err != nil {
		fmt.Fprintf(w, "There was an error while returning the result: %s\n ",
			err)
		log.Fatal(err)
	}
}

func SendRequestToBroker(op Operation) {
	// call to Broker API with address and the op jsonified
}

// ReadOperationFromRequest takes a request and a response to fill.
// Reads the request to extract the operation.
// Returns the operation from the request,
// writes an error in the response otherwise.
func ReadOperationFromRequest(r *http.Request, w http.ResponseWriter) Operation {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal([]byte(reqBody), &op)
	if err != nil {
		fmt.Fprintf(w, "There was an error reading the json: %s\n ", err)
		log.Fatal(err)
	}
	return op
}
