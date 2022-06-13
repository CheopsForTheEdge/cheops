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
	Index int 			`json:"Index"`
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
	key := utils.CreateResource(colnamerep, rep)
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

// GetReplicantAPI Gets a specific replicant from its meta ID
func GetReplicantAPI(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["MetaID"]
	rep, _ := GetReplicant(metaID)
	json.NewEncoder(w).Encode(rep)
	//w.WriteHeader(404)
}

// GetReplicant Gets a specific replicant from its meta ID
func GetReplicant(metaID string) (Replicant, string){
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

	replicant, key := GetReplicant(metaID)
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
	utils.SearchResource(colnamerep, "MetaID", id, &rep)
	if rep != nil {
		if rep.IsLeader {
			utils.DeleteResource(colnamerep, id)
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
	utils.SearchResource(colnamerep, "MetaID", id, &rep)
	if rep != nil {
		return true
	} else { return false }
}


func ExecuteReplication(op Operation, conf utils.Configurations) {
	if op.PlatformOperation == "create" {

		var resps []ExecutionResp

		// Executing operations on each sites
		// First, check if this is a redirection to know if we need to read sites
		if !(op.Redirection) {
			key := CreateReplicantFromOperation(op, true)
			fmt.Printf("Le replicant %s a été crée. \n", key)

			// Execute the request locally
			// TODO need threads (?) to execute the others in parallel
			stdout := ExecRequestLocally(op)
			fmt.Println(stdout)

			// Every operation becomes a redirection to avoid recursion
			op.Redirection = true

			// Send the operation to the broker for distribution
			// Formatting the call to the broker
			// for post, we need a reader, so we need the operation marshalled
			operation, _ := json.Marshal(op)
			opReader := strings.NewReader(string(operation))

			// the API to be called
			execAddress := "http://" + conf.LocalSite.Address + "/sendoperation"

			// Execute the actual request
			resp, err := http.Post(execAddress, "application/json",
				opReader)
				// Handle the error
			if err != nil {
				fmt.Printf("Error in executing command %s \n", execAddress)
				log.Fatal(err)
			}
				// Create the response and add it
			execResp := ExecutionResp{"site", "op.Request", *resp}
			resps = append(resps, execResp)

		} else { // An operation is a redirection, i.e. is not the leader
			// Create non leader replica
			key := CreateReplicantFromOperation(op, false)
			fmt.Printf("Le replicant %s a été crée. \n", key)
			// Execute the operation locally
			stdout := ExecRequestLocally(op)
			fmt.Println(stdout)
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


/*
_______  _______  _______ _________   _______ _________          _______  _______
(  ____ )(  ___  )(  ____ \\__   __/  (  ____ \\__   __/|\     /|(  ____ \(  ____ \
| (    )|| (   ) || (    \/   ) (     | (    \/   ) (   | )   ( || (    \/| (    \/
| (____)|| (___) || (__       | |     | (_____    | |   | |   | || (__    | (__
|     __)|  ___  ||  __)      | |     (_____  )   | |   | |   | ||  __)   |  __)
| (\ (   | (   ) || (         | |           ) |   | |   | |   | || (      | (
| ) \ \__| )   ( || )         | |     /\____) |   | |   | (___) || )      | )
|/   \__/|/     \||/          )_(     \_______)   )_(   (_______)|/       |/
*/

// AddLogToLeader
//goland:noinspection LanguageDetectionInspection
func AddLogToLeader(log string) int  {

	return 0
}


// TODO check index
// CheckReplicas Ensure that replicants are up-to-date
// Requires the id to be the leader of the replicants
func CheckReplicas(id string) []string {
	var outdated_sites []string
	var rep *Replicant
	// Getting the replicant, checking if it exists,
	// checking if it is the leader
	utils.SearchResource(colnamerep, "MetaID", id, &rep)
	if rep != nil {
		if rep.IsLeader {
			// We'll get all the replicants on every site and check it is equal
			// to the leader
			for _, replica := range rep.Replicas {
				var otherReplicant Replicant
				site := replica.Site
				//TODO: use broker API !!
				siteAddress := endpoint.GetSiteAddress(site)
				GetReplicantAPI := "http://" + siteAddress + ":8080" +
					"" + "/replicant/" + id
				resp, _ := http.Get(GetReplicantAPI)
				reqBody, _ := ioutil.ReadAll(resp.Body)
				err := json.Unmarshal([]byte(reqBody),
					&otherReplicant)
				if err != nil {
					fmt.Printf("There was an error retrieving the replicant" +
						" on site %s:\n %s. \n", site, err)
					log.Fatal(err)
				}
				if !reflect.DeepEqual(rep, otherReplicant) {
					outdated_sites = append(outdated_sites, site)
				}
			}
		}
	} else {
		fmt.Printf("There are no replicant with the identifier %s",	id)
		log.Fatal(rep)
	}
	return outdated_sites
}

// TODO later: leader election for Raft, right now,
// the leader is the first replicant