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
	for _ = range time.Tick(1 * time.Second) {
		resp, err := http.Get("http://localhost:8283/api/v1/pods")
		if err != nil {
			log.Println("error with kube")
		}
		io.Copy(os.Stderr, resp.Body)
		resp.Body.Close()
	}
}

func ensureProxyRunning(ctx context.Context) error {
	return exec.CommandContext(ctx, "kubectl", "proxy", "--port=8283").Start()
}

func GetPodsHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("http://localhost:8283/api/v1/pods")
	if err != nil {
		http.Error(w, "error with kube", http.StatusInternalServerError)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		http.Error(w, "error with kube", http.StatusInternalServerError)
	}

	http.ServeContent(w, r, "", time.Unix(0, 0), bytes.NewReader(buf))
}
