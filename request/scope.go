package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

var m = make(map[string]string)

type Scope struct {
	Sites []string `json:sites`
}

func RegisterRemoteSite(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "marche pas")
		return
	}
	json.Unmarshal(reqBody, &m)
	fmt.Fprintf(w, "Registered")
}

func GetRemoteSite(w http.ResponseWriter, r *http.Request) {
	site := mux.Vars(r)["site"]
	val := m[site]
	w.Write([]byte(val))
}

func IsPresent(arr []string, it string) bool {
	for i := 0; i < len(arr); i++ {
		if strings.Contains(arr[i], it) {
			return true
		}
	}
	return false
}

func GetRemoteIP(arr []string, it string) string {
	for i := 0; i < len(arr); i++ {
		if strings.Contains(arr[i], it) {
			start := strings.Index(arr[i], "/") + 1
			val := arr[i][start:]
			return m[val]
		}
	}
	return ""
}

// the IP 10.244.3.5 was the consul-server-0 IP at the time of this test, it needs to be changed according to the current setup
func Appb(w http.ResponseWriter, r *http.Request) {
	url := "http://10.244.2.2:8500/v1/catalog/service/app-b"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "curl/7.64.0")
	req.Header.Add("Accept", "*/*")
	client := &http.Client{}
	res, _ := client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)
	str := string(body)
	startAdd := strings.Index(str, "ServiceAddress") + 17
	sliceAdd := str[startAdd:]
	endAdd := strings.Index(sliceAdd, "\"") + startAdd
	serviceAdd := str[startAdd:endAdd]
	startPort := strings.Index(str, "ServicePort") + 13
	slicePort := str[startPort:]
	endPort := strings.Index(slicePort, ",") + startPort
	servicePort := str[startPort:endPort]
	myHeader := r.Header.Get("x-envoy-original-path")

	headers := r.Header
	scope, remote := headers["Scope"]
	if remote {
		remote = IsPresent(scope, "app-b")
	}

	if !remote {
		finalURL := "http://" + serviceAdd + ":" + servicePort + myHeader
		finalReq, _ := http.NewRequest("GET", finalURL, nil)
		finalClient := &http.Client{}
		finalRes, _ := finalClient.Do(finalReq)
		finalBody, _ := ioutil.ReadAll(finalRes.Body)
		w.Write(finalBody)
	} else {
		finalURL := "http://127.0.0.1:8080/SendRemote"
		finalReq, _ := http.NewRequest("GET", finalURL, nil)
		finalReq.Header.Add("service", "Appb")
		finalReq.Header.Add("destPath", myHeader)
		finalReq.Header.Add("destSite", GetRemoteIP(scope, "app-b"))
		finalClient := &http.Client{}
		finalRes, _ := finalClient.Do(finalReq)
		finalBody, _ := ioutil.ReadAll(finalRes.Body)
		w.Write(finalBody)
	}
}

// IP 172.16.97.1 is the IP of remote master node, must be changed to match setup
func SendRemote(w http.ResponseWriter, r *http.Request) {
	//url := "http://172.16.97.1:8081/HandleRemote"
	url := "http://" + r.Header.Get("destSite") + ":8081/HandleRemote"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("service", r.Header.Get("service"))
	req.Header.Add("destPath", r.Header.Get("destPath"))
	client := &http.Client{}
	res, _ := client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)
	w.Write(body)
}

func ExtractScope(w http.ResponseWriter, req *http.Request) {
	//Get the scope in the request Header : x-request-id
	var scopes = req.Header.Get("x-request-id")
	//Extract scopes
	Myscope := new(Scope)
	Myscope.Sites = append(Myscope.Sites, scopes)

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

func RedirectRequest(w http.ResponseWriter, req *http.Request) {
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

	if err != nil {
		log.Fatal(err)
	}

	//Do the Request to the good service
	resp, err := client.Do(proxyReq)
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

func TestAppC(w http.ResponseWriter, req *http.Request) {
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
	if err != nil {
		log.Fatal(err)
	}

	//Do the Request to the good service
	resp, err := client.Do(proxyReq)
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
func TestR(w http.ResponseWriter, req *http.Request) {

	client := http.Client{}

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

	testaddr, err := http.NewRequest("GET", "http://localhost:8080/GetAddr", bytes.NewReader(bodyBytes))
	testaddr.Header.Add("consulip", "10.244.4.3")
	testaddr.Header.Add("servicename", "app-b")
	resp, err := client.Do(testaddr)
	if err != nil {
		log.Fatal(err)
	}
	Textblock, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	Textblockstring := string(Textblock)

	json.NewEncoder(w).Encode(Textblockstring)

}

func GetAddr(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := http.Client{}

	// create a new url with the good scope

	url := fmt.Sprintf("http://%s:8500/v1/catalog/service/%s", req.Header.Get("consulip"), req.Header.Get("servicename"))
	proxyReq, err := http.NewRequest("GET", url, bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	//Do the Request to the good service
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	json.NewEncoder(w).Encode(req.Host)

	json.NewEncoder(w).Encode(req.URL)

	json.NewEncoder(w).Encode(resp.Header)

	json.NewEncoder(w).Encode(resp)
	fmt.Fprint(w, resp)
}
