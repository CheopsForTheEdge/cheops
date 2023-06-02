package operation

import (
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
	Site 	endpoint.Site `json:"Site"`
	ID 		string        `json:"ID"`
	Status  string        `json:"Status"`
	Logs    []Log         `json:"Logs"`
}

type Replicant struct {
	MetaID      string    `json:"MetaID"`
	Replicas	[]Replica `json:"Replicas"`
	Leader		string    `json:"Leader"`
}

type Log struct {
	Operation string    `json:"Operation"`
	Date time.Time		`json:"Date"`
}

// Test replicants (allReplicants and Replicants)
type allReplicants []Replicant


var Nantes = endpoint.Site{SiteName: "Nantes", Address: "127.0.0.1"}
var Paris = endpoint.Site{SiteName: "Paris", Address: "127.0.0.1"}


var Replicants = allReplicants{
	{
		MetaID:      utils.CreateMetaId(),
		Replicas:     []Replica{
			Replica{Site: Paris, ID: "65"},
			Replica{Site: Nantes, ID: "42"},
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
	rep.Leader = utils.Conf.LocalSite.SiteName
	key := utils.CreateResource(colnamerep, rep)
	return key
}

//CreateReplicantFromOperation Creates the first Replicant for the replicas
func CreateReplicantFromOperation(op Operation, leader string) string {
	rep := new(Replicant)
	rep.MetaID = utils.CreateMetaId()
	rep.Replicas = []Replica{}
	var site endpoint.Site
	var date = time.Now()
	for _, siteName := range op.Sites {
		site = endpoint.GetSite(siteName)
		rep.Replicas = append(rep.Replicas, Replica{Site: site,
													ID: "",
													Logs: []Log{Log{Operation: "creation",
																	Date: date}},
													})
	}
	rep.Leader = leader
	key := utils.CreateResource(colnamerep, rep)
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
	Leader := utils.Conf.LocalSite.SiteName
	key := CreateReplicantFromOperation(op, Leader)
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

// GetReplicantAPI Gets a specific replicant from its meta ID
func GetReplicantAPI(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["MetaID"]
	rep, _ := getReplicant(metaID)
	json.NewEncoder(w).Encode(rep)
	//w.WriteHeader(404)
}

// getReplicant Gets a specific replicant from its meta ID
func getReplicant(metaID string) (Replicant, string){
	var rep Replicant
	_, key := utils.SearchResource(colnamerep, "metaID", metaID, &rep)
	return rep, key
	//w.WriteHeader(404)
}

// GetAllReplicants Gets all replicants
//func GetAllReplicants(w http.ResponseWriter, r *http.Request) {
//	json.NewEncoder(w).Encode(Replicants)
//}

// AddReplica Add a replica to a replicant
func AddReplica(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["MetaID"]
	var addedReplica Replica

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter data: %s\n", err)
	}
	err = json.Unmarshal(reqBody, &addedReplica)
	if err != nil {
		fmt.Fprintf(w, "There was an error reading the json: %s\n ", err)
		return
	}

	replicant, key := getReplicant(metaID)
	replicant.Replicas = append(replicant.Replicas, addedReplica)

	utils.UpdateResource(colnamerep, key, replicant.Replicas)
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
	var op Operation
//	var w http.ResponseWriter
	var sites []string
	var isLeader bool
	utils.SearchResource(colnamerep, "MetaID", id, &rep)
	isLeader = (rep.Leader == utils.Conf.LocalSite.SiteName)
	siteaddress := conf.LocalSite.Address
	// using the ExecRequestLocally on each involved site
	execAddress := "http://" + siteaddress + ":8080" + "/operation" +
		"/sendoperation"
	if rep != nil {
		if isLeader {
			utils.DeleteResource(colnamerep, id)
			fmt.Printf("The replicant with ID %s has been deleted" +
				" successfully \n",	id)
			for _, replica := range rep.Replicas {
				if !isLeader {
					sites = append(sites, replica.Site.SiteName)
				}
			}
			op = Operation{Operation: "deleteResource",
				Sites: sites,
				Platform: "Cheops",
				Resource: "Replication",
				Instance: rep.MetaID,
				PlatformOperation: "DeleteReplicant",
				Request: "/replicant/" + id,
				Redirection: true,
			}
			operation, _ := json.Marshal(op)
			opReader := strings.NewReader(string(operation))
			// TODO manage err and resp
			http.Post(execAddress, "application/json",
				opReader)
			// client.SendOperationToSites(operation, w)
		}
	} else {
		sites = append(sites, rep.Leader)
		op = Operation{Operation: "deleteResource",
			Sites: sites,
			Platform: "Cheops",
			Resource: "Replication",
			Instance: rep.MetaID,
			PlatformOperation: "DeleteReplicant",
			Request: "/replicant/" + id,
			Redirection: true,
		}
		operation, _ := json.Marshal(op)
		opReader := strings.NewReader(string(operation))
		// TODO manage err and resp
		http.Post(execAddress, "application/json",
			opReader)
		// client.SendThisOperationToSites(op, w)
	}
}


// CheckIfReplicant Returns true if the id is in the database
func CheckIfReplicant(id string) bool {
	var rep *Replicant
	utils.SearchResource(colnamerep, "MetaID", id, &rep)
	return rep != nil
}


// CheckReplicasLog Checks if replicas are up to date
func CheckReplicasLog(id string) {
	var rep *Replicant
	utils.SearchResource(colnamerep, "MetaID", id, &rep)
	if rep != nil {
		if rep.Leader == utils.Conf.LocalSite.SiteName {
			if getLeader(*rep) != nil {
				for _, otherRep := range rep.Replicas {
					site := otherRep.Site
					// TODO: check logs directly, deepequal won't work
					//TODO: use API
					fmt.Printf("Checking replica on site %s \n", site.SiteName)
					//getReplicant := "http://" + siteAddress + ":8080" + "/replicant" +
					//	"/" + id
					// resp, _ := http.Get(getReplicant)
					//	otherRep = json.Unmarshal([]byte(resp.Body), &otherRep)
					reflect.DeepEqual(rep, otherRep)
				}
			}
		} else {
			fmt.Println("The leader is not on this site")
		}
	} else {
		fmt.Println("The replicant does not exists")
		log.Fatal(rep)
	}
	//TODO: return a list of NEQUAL replicas?
}


// getLeader Returns the leader of a replicant
func getLeader (replicant Replicant) *Replica {
	var replica Replica
	for _, rep := range replicant.Replicas {
		if rep.Site.SiteName == replicant.Leader {
			return &rep
		}
	}
	fmt.Println("The leader was not found")
	log.Fatal(replica)
	return nil
}

func ExecuteReplication(op Operation, conf utils.Configurations) {
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
			// Every operation becomes a redirection to avoid recursion
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
			if CheckIfReplicant(op.Instance) {
				DeleteReplicantWithID(op.Instance)
			}
		}
	}
}