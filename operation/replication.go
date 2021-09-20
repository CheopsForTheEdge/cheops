package operation

import (
	"cheops.com/database"
	"cheops.com/utils"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"time"
)

type Replica struct {
	Site 	string `json:"Site"`
	ID 		string `json:"ID"`
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
var colnamerep = "replication"




// CreateReplicant Creates a replicant with a meta ID, probably needs to add also the locations
func CreateReplicant() string {
	rep := new(Replicant)
	rep.MetaID = utils.CreateMetaId()
	rep.Replicas = []Replica{}
	rep.IsLeader = true
	rep.Logs = []Log{
		Log{Operation: "creation", Date: (time.Now())}}
	key := database.CreateResource(colnamerep, rep)
	return key
}

//CreateLeaderFromOperation Creates the first Replicant for the replicas
func CreateLeaderFromOperation(op Operation) string {
	rep := new(Replicant)
	rep.MetaID = utils.CreateMetaId()
	rep.Replicas = []Replica{}
	fmt.Println(op.Sites)
	for _, site := range op.Sites{
		rep.Replicas = append(rep.Replicas, Replica{Site: site, ID:""})
	}
	rep.IsLeader = true
	rep.Logs = []Log{
		Log{Operation: op.Request, Date: (time.Now())}}
	key := database.CreateResource(colnamerep, rep)
	return key
}

func CreateLeaderFromOperationAPI(w http.ResponseWriter, r *http.Request) {
	var op Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(reqBody, &op)
	fmt.Println(op)
	key := CreateLeaderFromOperation(op)
	json.NewEncoder(w).Encode(key)
}


// CreateReplicantFromUID Creates a replicant with given information
func CreateReplicantFromUID(w http.ResponseWriter, r *http.Request)  {
	var newReplicant Replicant
	reqBody, _ := ioutil.ReadAll(r.Body)

	json.Unmarshal(reqBody, &newReplicant)
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
		fmt.Fprintf(w, "Kindly enter data")
	}
	json.Unmarshal(reqBody, &updatedReplicant)

	for i, rep := range Replicants {
		if rep.MetaID == metaID {
			var replica Replica
			var numberReplicas = len(rep.Replicas)
			replica.Site = updatedReplicant.Replicas[numberReplicas].Site
			replica.ID = updatedReplicant.Replicas[numberReplicas].Site
			rep.Replicas = append(rep.Replicas, replica)
			Replicants = append(Replicants[:i], rep)
			json.NewEncoder(w).Encode(rep)
		}
	}
}

// DeleteReplicant Deletes a replicant given a meta ID
func DeleteReplicant(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["MetaID"]
	for i, rep := range Replicants {
		if rep.MetaID == metaID {
			Replicants = append(Replicants[:i], Replicants[i+1:]...)
			fmt.Fprintf(w,
				"The event with ID %v has been deleted successfully",
				metaID)
		}
	}
}

// DeleteReplicant Deletes a replicant given a meta ID
func DeleteReplicantWithKey(key string) {
	database.DeleteResource(colnamerep, key)
	fmt.Printf("The event with ID %s has been deleted successfully \n", key)
}
