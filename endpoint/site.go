package endpoint

import (
	"cheops.com/database"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

// Site is the global endpoint for the site's Cheops
type Site struct {
	SiteName		string  `json:"SiteName"`
	Address 		string  `json:"Address"`
}


// Collection name variable
var colnamesite = "sites"


func CreateSite(siteName string, address string) string {
	end := Site{SiteName: siteName, Address: address}
	return database.CreateResource(colnamesite, end)
}

func CreateSiteAPI(w http.ResponseWriter, r *http.Request) {
	site := mux.Vars(r)["Site"]
	add := mux.Vars(r)["Address"]
	key := CreateSite(site, add)
	json.NewEncoder(w).Encode(key)
}

func GetSite(siteName string) Site {
	query := "FOR end IN sites FILTER end.Site == @name RETURN end"
	bindvars := map[string]interface{}{ "name": siteName}
	result := Site{}
	database.ExecuteQuery(query, bindvars, &result)
	if &result == nil {
		err := fmt.Sprintf("Address %s not found.\n", siteName)
		fmt.Print(err)
		log.Fatal(err)
	}
	return result
}

func GetSiteAddress(siteName string) string {
	query := "FOR end IN endpoint FILTER end.Site == @name RETURN end"
	bindvars := map[string]interface{}{ "name": siteName}
	result := Endpoint{}
	database.ExecuteQuery(query, bindvars, &result)
	if result.Address == "" {
		err := fmt.Sprintf("Address %s not found.\n", siteName)
		fmt.Print(err)
		log.Fatal(err)
	}
	return result.Address
}


func GetSiteAddressAPI(w http.ResponseWriter, r *http.Request) {
	site := mux.Vars(r)["Site"]
	add := GetSiteAddress(site)
	if add != "" {
		json.NewEncoder(w).Encode(add)
		return
	}
	w.WriteHeader(404)
}