package api

import(
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"../replication"
)


func Routing() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeLink)
	fmt.Printf("%v replicants", replication.Replicants)
	commonHandlers := alice.New(CheckRequestFilledHandler)
	router.Handle("/replication", commonHandlers.ThenFunc(replication.CreateReplicant)).Methods("POST")
	router.HandleFunc("/replicant/{metaID}", replication.GetReplicant).Methods("GET")
	router.HandleFunc("/replicant/{metaID}", replication.AddReplica).Methods("PUT")
	router.HandleFunc("/replicant/{metaID}", replication.DeleteReplicant).Methods("DELETE")
	router.Handle("/replicants", commonHandlers.ThenFunc(replication.
		GetAllReplicants)).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", router))
}


func CheckRequestFilledHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		_, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Kindly enter data", r.Method, r.URL.String())

			fmt.Fprintf(w, "Kindly enter data")
			return
		} else {
			next.ServeHTTP(w, r)
		}
	}
	return http.HandlerFunc(fn)
}


func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome home!")
}