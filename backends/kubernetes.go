package backends

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func Kubernetes(ctx context.Context) {
	err := ensureProxyRunning(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func ensureProxyRunning(ctx context.Context) error {
	return exec.CommandContext(ctx, "kubectl", "proxy", "--port=8283").Start()
}

func SitesFor(method string, path string, headers http.Header, body []byte) ([]string, error) {
	if body == nil || len(body) == 0 {
		return make([]string, 0), nil
	}

	doc, err := yaml.Parse(string(body))
	if err != nil {
		return nil, err
	}

	meta, err := doc.GetMeta()
	if err != nil {
		return nil, err
	}
	locationsString, ok := meta.ObjectMeta.Annotations["locations"]
	if !ok {
		return make([]string, 0), nil
	}
	locations := strings.Split(locationsString, ",")
	locTrimmed := make([]string, 0)
	for _, loc := range locations {
		locTrimmed = append(locTrimmed, strings.TrimSpace(loc))
	}

	return locTrimmed, nil
}

func HandleKubernetes(method string, path string, headers http.Header, body []byte) (http.Header, []byte, error) {
	u := fmt.Sprintf("http://127.0.0.1:8283%s", path)

	newreq, err := http.NewRequestWithContext(context.Background(), method, u, bytes.NewReader(body))

	if err != nil {
		return nil, nil, err
	}

	for key, vals := range headers {
		for _, val := range vals {
			newreq.Header.Add(key, val)
		}
	}

	resp, err := http.DefaultClient.Do(newreq)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respbuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	headersOut := make(http.Header)
	for key, vals := range resp.Header {
		for _, val := range vals {
			headersOut.Add(key, val)
		}
	}

	return headersOut, respbuf, nil
}
