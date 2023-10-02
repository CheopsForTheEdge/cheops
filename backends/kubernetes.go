package backends

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func SitesFor(method string, path string, headers http.Header, body []byte) ([]string, error) {
	if body == nil || len(body) == 0 {
		return make([]string, 0), nil
	}

	doc, err := yaml.Parse(string(body))
	if err != nil {
		return nil, err
	}

	locationsMap := doc.GetAnnotations("locations")
	if len(locationsMap) == 0 {
		return make([]string, 0), nil
	}
	locations := strings.Split(locationsMap["locations"], ",")
	locTrimmed := make([]string, 0)
	for _, loc := range locations {
		locTrimmed = append(locTrimmed, strings.TrimSpace(loc))
	}

	return locTrimmed, nil
}

func HandleKubernetes(ctx context.Context, method string, path string, headers http.Header, body []byte) (h http.Header, b []byte, err2 error) {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	command := cmd.String()
	log.Printf("Running %s\n", command)
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
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		log.Printf("[%s]: %s\n", command, scanner.Text())
	}
	return h, out, err
}

// CurrentConfig fetches the configuration as json for the resources that is given in input.
// The input is useful for determining the namespace and name. We let kubectl do the
// magic itself here.
// If anything goes wrong, CurrentConfig returns an empty json object
func CurrentConfig(ctx context.Context, targetResource []byte) []byte {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "-o", "yaml", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Couldn't get stdin pipe: %v\n", err)
		return []byte("{}")
	}

	go func() {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewBuffer(targetResource))
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Couldn't run command: %v\n", err)
		return []byte("{}")
	}

	return extractCurrentConfig(out)
}

func extractCurrentConfig(b []byte) []byte {
	node, err := yaml.Parse(string(b))
	if err != nil {
		log.Printf("Couldn't parse yaml output: %v\n", err)
		return []byte("{}")
	}
	docs, err := node.Pipe(yaml.Lookup("items"))
	if err == nil {
		elements, err := docs.Elements()
		if err != nil {
			log.Printf("Error parsing config: %v\n", err)
			return []byte("{}")

		}
		node = elements[0]
	}

	m := node.GetAnnotations("kubectl.kubernetes.io/last-applied-configuration")
	config, ok := m["kubectl.kubernetes.io/last-applied-configuration"]
	if !ok {
		return []byte("{}")
	}
	return []byte(config)
}
