package backends

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// runWithStdin runs a command with an input to be passed to standard input and returns the combined output (stdout and stderr) as a slice of bytes and an error.
//
// If the command was run successfully with no error, err is null.
// If the command was run successfully with a status code != 0, the error is a generic "failed". Note that stderr is included in the output
// If there was an error in running the command (command not found, ...) then the error is a generic "internal error". The underlying error is logged.
func runWithStdin(ctx context.Context, input string, args ...string) (output string, err error) {
	req := strings.Split(input, " ")
	if len(req) == 0 {
		return "", fmt.Errorf("Invalid command")
	}
	cmd := exec.CommandContext(ctx, req[0], req[1:]...)
	command := cmd.String()
	log.Printf("Running %s\n", command)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Couldn't run [%s]: %v\n", command, err)
		return "", fmt.Errorf("internal error")
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		log.Printf("[%s]: %s\n", command, scanner.Text())
	}

	if cmd.ProcessState != nil && !cmd.ProcessState.Success() {
		return "", fmt.Errorf("failed")
	}

	return string(out), nil
}

func Handle(ctx context.Context, bodies []string) (replies []string, err error) {
	replies = make([]string, 0)
	doRun := true

	for _, body := range bodies {
		var output string
		if doRun {
			out, err2 := runWithStdin(ctx, body, "sh", "-c")
			if err2 != nil {
				err = err2
				doRun = false
			}
			output = out
		}
		replies = append(replies, output)
	}
	return replies, err
}

func DeleteKubernetes(ctx context.Context, input []byte) error {
	_, err := runWithStdin(ctx, string(input), "kubectl", "delete", "-f", "-")
	return err
}
