package request

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestAppC (w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := http.Client{}

	// create a new url with the good scope
	ScopeAddr := "http://10.244.1.8:5002/resourceb/1"
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
	//fmt.Fprint(w, resp.Body)
	//fmt.Fprint(w,req.Header)
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bodyString := string(bodyBytes)
	json.NewEncoder(w).Encode(bodyString)
	json.NewEncoder(w).Encode(resp.Header)



}
func TestR (w http.ResponseWriter, req *http.Request) {

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bodyString := string(bodyBytes)
	json.NewEncoder(w).Encode(req.Host)
	json.NewEncoder(w).Encode(req.URL)
	json.NewEncoder(w).Encode(req.Header)
	json.NewEncoder(w).Encode(bodyString)

	json.NewEncoder(w).Encode(GetAddr("10.244.4.3", "app-b"))



}


func GetAddr(consulip string, servicename string) string {


	var w http.ResponseWriter
	var req *http.Request
	client := http.Client{}


	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return ""
	}
	url := fmt.Sprintf("%s:8500/v1/catalog/service/%s", consulip, servicename)
	curlReq, err := http.NewRequest("GET", url,bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return ""
	}

	resp , err := client.Do(curlReq)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyresp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return ""
	}
	bodyString := string(bodyresp)
	return bodyString
}
