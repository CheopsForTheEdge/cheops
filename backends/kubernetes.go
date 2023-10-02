package backends

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func ResourceIdFor(method string, path string, headers http.Header, body []byte) (string, error) {
	if body == nil || len(body) == 0 {
		return "", fmt.Errorf("No body to give back an id")
	}

	doc, err := yaml.Parse(string(body))
	if err != nil {
		return "", err
	}

	kind := doc.GetKind()

	meta, err := doc.GetMeta()
	if err != nil {
		return "", err
	}
	name := meta.ObjectMeta.NameMeta.Name
	namespace := meta.ObjectMeta.NameMeta.Namespace
	if namespace == "" {
		namespace = "default"
	}

	return fmt.Sprintf("%s:%s:%s", namespace, kind, name), nil
}

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

func runWithStdin(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	command := cmd.String()
	log.Printf("Running %s\n", command)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	go func() {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewBuffer(input))
	}()

	out, err := cmd.CombinedOutput()
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		log.Printf("[%s]: %s\n", command, scanner.Text())
	}
	return out, err
}

func HandleKubernetes(ctx context.Context, method string, path string, headers http.Header, input []byte) (header http.Header, body []byte, err error) {
	out, err := runWithStdin(ctx, input, "kubectl", "apply", "-f", "-")
	return header, out, err
}

// CurrentConfig fetches the configuration as json for the resources that is given in input.
// The input is useful for determining the namespace and name. We let kubectl do the
// magic itself here.
// If anything goes wrong, CurrentConfig returns an empty json object
func CurrentConfig(ctx context.Context, targetResource []byte) []byte {
	out, err := runWithStdin(ctx, targetResource, "kubectl", "get", "-o", "yaml", "-f", "-")
	if err != nil {
		log.Printf("Error running cmd: %s\n", err)
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
