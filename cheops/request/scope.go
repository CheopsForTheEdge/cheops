package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Scope struct {
	Sites []string `json:sites`
}

//the IP 10.244.3.5 was the consul-server-0 IP at the time of this test, it needs to be changed according to the current setup
func Appb (w http.ResponseWriter, r *http.Request) {
        url := "http://10.244.3.5:8500/v1/catalog/service/app-b"
        req, _ := http.NewRequest("GET", url, nil)
        req.Header.Add("User-Agent", "curl/7.64.0")
        req.Header.Add("Accept", "*/*")
        client := &http.Client{}
        res, _ := client.Do(req)
        body, _ := ioutil.ReadAll(res.Body)
        str := string(body)
        startAdd := strings.Index(str, "ServiceAddress") + 17
        sliceAdd := str[startAdd : ]
        endAdd := strings.Index(sliceAdd, "\"") + startAdd
        serviceAdd := str[startAdd : endAdd]
        startPort := strings.Index(str, "ServicePort") + 13
        slicePort := str[startPort : ]
        endPort := strings.Index(slicePort, ",") + startPort
        servicePort := str[startPort : endPort]
        myHeader := r.Header.Get("x-envoy-original-path")

        finalURL := "http://" + serviceAdd + ":" + servicePort + myHeader
        finalReq, _ := http.NewRequest("GET", finalURL, nil)
        finalClient := &http.Client{}
        finalRes, _ := finalClient.Do(finalReq)
        finalBody, _ := ioutil.ReadAll(finalRes.Body)
        finalStr := string(finalBody)
        json.NewEncoder(w).Encode(finalStr)
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
