package replication

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

type Replica struct {
	Site 	string `json:"site"`
	ID 		string `json:"ID"`
}

type Replicant struct {
	MetaID      string    `json:"ID"`
	Replicas	[]Replica `json:"replicas"`
}

// Test replicants (allReplicants and Replicants)
type allReplicants []Replicant

var Replicants = allReplicants{
	{
		MetaID:      "33344596",
		Replicas:     []Replica{
			Replica{Site: "Paris", ID: "65"},
			Replica{Site: "Nantes", ID: "42"},
			},
	},
}

// Creates a replicant with a meta ID, probably needs to add also the locations
func CreateReplicant(w http.ResponseWriter, r *http.Request) {
	rep := new(Replicant)
	rep.MetaID = string(42) // TODO: generate MetaUID
	rep.Replicas = []Replica{}
	Replicants = append(Replicants, *rep)
	json.NewEncoder(w).Encode(Replicants)
}

// Creates a replicant with given information
func CreateReplicantFromUID(w http.ResponseWriter, r *http.Request)  {
	var newReplicant Replicant
	reqBody, _ := ioutil.ReadAll(r.Body)

	json.Unmarshal(reqBody, &newReplicant)
	Replicants = append(Replicants, newReplicant)
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(newReplicant)
}

// Gets a specific replicant from its meta ID
func GetReplicant(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["metaID"]

	for _, rep := range Replicants {
		if rep.MetaID == metaID {
			json.NewEncoder(w).Encode(rep)
			return
		}
	}
	w.WriteHeader(404)
}

// Gets all replicants
func GetAllReplicants(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Replicants)
}

// Add a replica to a replicant
func AddReplica(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["metaID"]
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

// Deletes a replicant given a meta ID
func DeleteReplicant(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["metaID"]
	for i, rep := range Replicants {
		if rep.MetaID == metaID {
			Replicants = append(Replicants[:i], Replicants[i+1:]...)
			fmt.Fprintf(w,
				"The event with ID %v has been deleted successfully",
				metaID)
		}
	}
}