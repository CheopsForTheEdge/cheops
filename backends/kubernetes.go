package backends

import (
	"bytes"
	"context"
	"io"
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

func HandleKubernetes(ctx context.Context, method string, path string, headers http.Header, body []byte) (h http.Header, b []byte, err2 error) {
	cmd := exec.CommandContext(ctx, "kubectl", "--server-side=true", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		err2 = err
		return
	}

	go func() {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewBuffer(body))
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		err2 = err
		return
	}

	return h, out, nil
}

// CurrentConfig fetches the configuration as json for the resources that is given in input.
// The input is useful for determining the namespace and name. We let kubectl do the
// magic itself here.
// If anything goes wrong, CurrentConfig returns an empty json object
func CurrentConfig(ctx context.Context, targetResource []byte) []byte {
	cmd := exec.CommandContext(ctx, "kubectl", "-o", "yaml", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return []byte("{}")
	}

	go func() {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewBuffer(targetResource))
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return []byte("{}")
	}

	return out
}
