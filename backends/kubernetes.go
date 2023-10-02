package backends

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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

// runWithStdin runs a command with an input to be passed to standard input and returns the combined output (stdout and stderr) as a slice of bytes and an error. If the length of input is 0 then the command is run as-is.
//
// If the command was run successfully with no error, err is null.
// If the command was run successfully with a status code != 0, the error is a generic "failed". Note that stderr is included in the output
// If there was an error in running the command (command not found, ...) then the error is a generic "internal error". The underlying error is logged.
func runWithStdin(ctx context.Context, input []byte, args ...string) (output []byte, err error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	command := cmd.String()
	log.Printf("Running %s\n", command)

	if len(input) > 0 {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Printf("Couldn't get stdinpipe for [%s]: %v\n", command, err)
			return nil, fmt.Errorf("internal error")
		}

		go func() {
			defer stdin.Close()
			io.Copy(stdin, bytes.NewBuffer(input))
		}()
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Couldn't run [%s]: %v\n", command, err)
		return out, fmt.Errorf("internal error")
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		log.Printf("[%s]: %s\n", command, scanner.Text())
	}

	if cmd.ProcessState != nil && !cmd.ProcessState.Success() {
		return out, fmt.Errorf("failed")
	}

	return out, nil
}

func HandleKubernetes(ctx context.Context, method string, path string, headers http.Header, input []byte) (header http.Header, body []byte, err error) {
	out, err := runWithStdin(ctx, input, "kubectl", "apply", "-f", "-")
	return header, out, err
}

// CurrentConfig fetches the configuration as json for the resources that is given in input.
// The input is useful for determining the namespace and name. We let kubectl do the
// magic itself here.
// If anything goes wrong, CurrentConfig returns an empty json object
func CurrentConfig(ctx context.Context, targetId string) []byte {
	out, err := runWithStdin(ctx, nil, "kubectl", "get", "-o", "json", "-f", "-")
	if err != nil {
		log.Printf("Error running cmd: %s\n", err)
		return []byte("{}")
	}

	return extractCurrentConfig(out, targetId)
}

func extractCurrentConfig(b []byte, targetId string) []byte {
	var ac allConfig
	err := json.Unmarshal(b, &ac)
	if err != nil {
		log.Printf("Couldn't parse yaml output: %v\n", err)
		return []byte("{}")
	}

	for _, item := range ac.Items {
		namespace := item.Metadata.NameMeta.Namespace
		if namespace == "" {
			namespace = "default"
		}
		id := fmt.Sprintf("%s:%s:%s", namespace, item.Kind, item.Metadata.NameMeta.Name)
		if id == targetId {
			lastAppliedConfiguration, ok := item.Metadata.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			fmt.Println(lastAppliedConfiguration)
			fmt.Println(ok)
			if ok {
				return []byte(lastAppliedConfiguration)
			}
		}
	}

	return []byte("{}")
}

type allConfig struct {
	Items []itemConfig `json:"items"`
}

type itemConfig struct {
	Kind     string          `json:"kind"`
	Metadata yaml.ObjectMeta `json:"metadata"`
}
