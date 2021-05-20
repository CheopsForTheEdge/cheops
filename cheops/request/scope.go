package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type Scope struct {
	Sites []string `json:sites`
}

func ExtractScope(w http.ResponseWriter, req *http.Request) {
	//Get the scope in the request Header : x-request-id
	var scopes = req.Header.Get("x-request-id")
	//Extract scopes
	Myscope := new(Scope)
	Myscope.Sites= append(Myscope.Sites, scopes)

	////Test to see if forward is working
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bodyString := string(bodyBytes)

	json.NewEncoder(w).Encode(Myscope)
	fmt.Fprint(w, bodyString)
	fmt.Fprint(w, req.Header)


}

func RedirectRequest (w http.ResponseWriter, req *http.Request) {
	//Check if the incoming body is nil or not
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := http.Client{}

	// create a new url with the good scope
	ScopeAddr := "http://localhost:8080/scope"
	url := fmt.Sprintf("%s", ScopeAddr)
	proxyReq, err := http.NewRequest("GET", url, bytes.NewReader(body))
	proxyReq.Header = req.Header

	if err  != nil{
		log.Fatal(err)
	}

	//Do the Request to the good service
	resp , err := client.Do(proxyReq)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	//Test to see if forward is working
	bodyt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bodyString := string(bodyt)
	fmt.Fprint(w, resp.Header.Get("x-request-id"))
	fmt.Fprint(w, bodyString)



}

func GetScopeAddr() {

}
