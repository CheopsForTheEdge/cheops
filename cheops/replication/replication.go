package replication

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

type Replica struct {
	site 	string `json:"site"`
	ID 		string `json:"ID"`
}

type Replicant struct {
	metaID      string    `json:"ID"`
	replicas	[]Replica `json:"replicas"`
}

type allReplicants []Replicant

// var replicaParis = Replica{site: "Paris", ID: "65"}
// var replicaNantes = Replica{site: "Nantes", ID: "42"}

var Replicants = allReplicants{
	{
		metaID:      "33344596",
		replicas:     []Replica{
			Replica{site: "Paris", ID: "65"},
			Replica{site: "Nantes", ID: "42"},
			},
	},
}


func CreateReplicant(w http.ResponseWriter, r *http.Request) {
	var newReplicant Replicant
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter data")
		return
	}

	json.Unmarshal(reqBody, &newReplicant)
	Replicants = append(Replicants, newReplicant)
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(newReplicant)
}

func GetReplicant(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["metaID"]

	for _, rep := range Replicants {
		if rep.metaID == metaID {
			json.NewEncoder(w).Encode(rep)
			return
		}
	}
	w.WriteHeader(404)
}

// this doesn't even work...
func GetAllReplicants(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Replicants)
	// fmt.Println(json.NewEncoder(w).Encode(Replicants))
	// fmt.Fprintf(w, "replicants : %v", Replicants)
}

func AddReplica(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["metaID"]
	var updatedReplicant Replicant

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter data")
	}
	json.Unmarshal(reqBody, &updatedReplicant)

	for i, rep := range Replicants {
		if rep.metaID == metaID {
			var replica Replica
			var numberReplicas = len(rep.replicas)
			replica.site = updatedReplicant.replicas[numberReplicas].site
			replica.ID = updatedReplicant.replicas[numberReplicas].site
			rep.replicas = append(rep.replicas, replica)
			Replicants = append(Replicants[:i], rep)
			json.NewEncoder(w).Encode(rep)
		}
	}
}


func DeleteReplicant(w http.ResponseWriter, r *http.Request) {
	metaID := mux.Vars(r)["metaID"]
	for i, rep := range Replicants {
		if rep.metaID == metaID {
			Replicants = append(Replicants[:i], Replicants[i+1:]...)
			fmt.Fprintf(w,
				"The event with ID %v has been deleted successfully",
				metaID)
		}
	}
}