package operation

import (
	"cheops.com/config"
	"cheops.com/database"
	"cheops.com/endpoint"
	"cheops.com/utils"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type Replica struct {
	Site 	string `json:"Site"`
	ID 		string `json:"ID"`
	Status  string `json:"Status"`
}

type Replicant struct {
	MetaID      string    `json:"MetaID"`
	Replicas	[]Replica `json:"Replicas"`
	IsLeader	bool      `json:"IsLeader"`
	Logs        []Log     `json:"Logs"`
}

type Log struct {
	Operation string    `json:"Operation"`
	Date time.Time		`json:"Date"`
}

// Test replicants (allReplicants and Replicants)
type allReplicants []Replicant

var Replicants = allReplicants{
	{
		MetaID:      utils.CreateMetaId(),
		Replicas:     []Replica{
			Replica{Site: "Paris", ID: "65"},
			Replica{Site: "Nantes", ID: "42"},
		},
	},
}

// Collection name variable
var colnamerep = "replications"


// CreateReplicant Creates a replicant with a meta ID, probably needs to add also the locations
func CreateReplicant() string {
	rep := new(Replicant)
	rep.MetaID = utils.CreateMetaId()
	rep.Replicas = []Replica{}
	rep.IsLeader = true
	rep.Logs = []Log{
		Log{Operation: "creation", Date: time.Now()}}
	key := database.CreateResource(colnamerep, rep)
	return key
}

//CreateReplicantFromOperation Creates the first Replicant for the replicas
func CreateReplicantFromOperation(op Operation, isLeader bool) string {
	rep := new(Replicant)
	rep.MetaID = utils.CreateMetaId()
	rep.Replicas = []Replica{}
	for _, site := range op.Sites{
		rep.Replicas = append(rep.Replicas, Replica{Site: site, ID:""})
	}
	rep.IsLeader = isLeader
	rep.Logs = []Log{
		Log{Operation: op.Request, Date: time.Now()}}
	key := database.CreateResource(colnamerep, rep)
	return key
}

func CreateReplicantFromOperationAPI(w http.ResponseWriter, r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal([]byte(reqBody), &op)
	if err != nil {
		fmt.Fprintf(w, "There was an error reading the json: %s\n ", err)
		return
	}
	conf := utils.GetConfig()
	isLeader := (conf.Site == op.Sites[0])
	key := CreateReplicantFromOperation(op, isLeader)
	json.NewEncoder(w).Encode(key)
}


// CreateReplicantAPI Creates a replicant with given information
func CreateReplicantAPI(w http.ResponseWriter, r *http.Request)  {
	var newReplicant Replicant
	reqBody, _ := ioutil.ReadAll(r.Body)

	err:= json.Unmarshal(reqBody, &newReplicant)
	if err != nil {
		fmt.Fprintf(w, "There was an error reading the json: %s\n ", err)
		return
	}
	Replicants = append(Replicants, newReplicant)
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(newReplicant)
}

// GetReplicant Gets a specific replicant from its meta ID
func GetReplicant(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["MetaID"]

	for _, rep := range Replicants {
		if rep.MetaID == metaID {
			json.NewEncoder(w).Encode(rep)
			return
		}
	}
	w.WriteHeader(404)
}

// GetAllReplicants Gets all replicants
func GetAllReplicants(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Replicants)
}

// AddReplica Add a replica to a replicant
func AddReplica(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["MetaID"]
	var updatedReplicant Replicant

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter data: %s\n", err)
	}
	err = json.Unmarshal(reqBody, &updatedReplicant)
	if err != nil {
		fmt.Fprintf(w, "There was an error reading the json: %s\n ", err)
		return
	}

	for i, rep := range Replicants {
		if rep.MetaID == metaID {
			var replica Replica
			var numberReplicas = len(rep.Replicas)
			replica.Site = updatedReplicant.Replicas[numberReplicas].Site
			replica.ID = updatedReplicant.Replicas[numberReplicas].Site
			rep.Replicas = append(rep.Replicas, replica)
			Replicants = append(Replicants[:i], rep)
			err := json.NewEncoder(w).Encode(rep)
			if err != nil {
				fmt.Fprintf(w, "There was an error reading the json: %s\n ", err)
				return
			}
		}
	}
}

// DeleteReplicant Deletes a replicant given a meta ID
func DeleteReplicant(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["MetaID"]
	DeleteReplicantWithID(metaID)
	fmt.Fprintf(w, "The replicant with ID %v has been deleted successfully",
		metaID)
}

// DeleteReplicantWithID Deletes a replicant given a meta ID
func DeleteReplicantWithID(id string) {
	var rep *Replicant
	database.SearchResource(colnamerep, "MetaID", id, &rep)
	if rep != nil {
		if rep.IsLeader {
			database.DeleteResource(colnamerep, id)
			fmt.Printf("The event with ID %s has been deleted successfully \n", id)
			for _, replica := range rep.Replicas {
				site := replica.Site
				//TODO: use API + send to broker
				siteAddress := endpoint.GetSiteAddress(site)
				getReplicant := "http://" + siteAddress + ":8080" + "/replicant" +
					"/" + id
				//TODO: maybe do something with the result
				http.NewRequest("DELETE", getReplicant, nil)
			}
		} else {
			//TODO: send the request to the leader
		}
	}

}

// CheckIfReplicant Returns true if the id is in the database
func CheckIfReplicant(id string) (isReplicant bool) {
	var rep *Replicant
	database.SearchResource(colnamerep, "MetaID", id, &rep)
	if rep != nil {
		return true
	} else { return false }
}

// CheckReplicas Requires the id to be the leader of the replicants
func CheckReplicas(id string) {
	var rep *Replicant
	database.SearchResource(colnamerep, "MetaID", id, &rep)
	if rep != nil {
		//TODO: maybe check if leader to be sure
		for _, replica := range rep.Replicas {
			var otherRep Replicant
			site := replica.Site
			//TODO: use API
			endpoint.GetSiteAddress(site)
			//getReplicant := "http://" + siteAddress + ":8080" + "/replicant" +
			//	"/" + id
			// resp, _ := http.Get(getReplicant)
			//	otherRep = json.Unmarshal([]byte(resp.Body), &otherRep)
			reflect.DeepEqual(rep, otherRep)
		}
	} else {
		fmt.Println("The replicant does not exists")
		log.Fatal(rep)
	}
	//TODO: return a list of NEQUAL replicas?
}

func ExecuteReplication(op Operation, conf config.Configurations) {
	if op.PlatformOperation == "create" {
		// TODO: cf notebook
		//replicationAdd := "http://" + siteadd + ":8080" + "/replication"
		//resp, _ = http.Post(replicationAdd, "application/json", opReader)
		//if resp != nil {
		//	execResp = ExecutionResp{"site", "createReplicant", *resp}
		//	resps = append(resps, execResp)
		//}
		var resps []ExecutionResp
		// Executing operations on each sites, might need threads to do it in parallel
		for _, site := range op.Sites {
			siteaddress := endpoint.GetSiteAddress(site)
			// using the ExecRequestLocally on each involved site
			execAddress := "http://" + siteaddress + ":8080" + "/operation" +
				"/localrequest"

			//
			op.Redirection = true
			// for post, we need a reader, so we need the operation marshalled
			operation, _ := json.Marshal(op)
			opReader := strings.NewReader(string(operation))
			// execute the actual request
			// TODO: ExecRequestLocallyAPI for the broker
			resp, err := http.Post(execAddress, "application/json",
				opReader)
			
			if err != nil {
				fmt.Printf("Error in executing command %s \n", execAddress)
				log.Fatal(err)
			}
			// create the response
			execResp := ExecutionResp{"site", "op.Request", *resp}
			resps = append(resps, execResp)
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