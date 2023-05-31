package kubernetes

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func Run(ctx context.Context) {
	err := ensureProxyRunning(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if false {
		for _ = range time.Tick(1 * time.Second) {
			resp, err := http.Get("http://localhost:8283/api/v1/pods")
			if err != nil {
				log.Println("error with kube")
			}
			io.Copy(os.Stderr, resp.Body)
			resp.Body.Close()
		}
	}
}

func ensureProxyRunning(ctx context.Context) error {
	return exec.CommandContext(ctx, "kubectl", "proxy", "--port=8283").Start()
}

func Proxy(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	reqbuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read request body", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	u := r.URL
	u.Host = "127.0.0.1:8283"

	// Not filled by default
	u.Scheme = "http"

	newreq, err := http.NewRequestWithContext(context.Background(), r.Method, u.String(), bytes.NewReader(reqbuf))
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "can't build proxy request", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	resp, err := http.DefaultClient.Do(newreq)
	if err != nil {
		http.Error(w, "can't send to backend", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	respbuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "can't read reply from backend", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	for key, vals := range resp.Header {
		for _, val := range vals {
			w.Header().Add(key, val)
		}
	}

	_, err = io.Copy(w, bytes.NewReader(respbuf))
	if err != nil {
		http.Error(w, "can't write reply", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	log.Printf("-> %s\n<- %s\n", r.URL.String(), string(respbuf))
}
