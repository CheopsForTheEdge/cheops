package backends

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
)

// A Command is a command with the files it needs
type ShellCommand struct {
	Command string
	Files   map[string][]byte
}

// runWithStdin runs a command with an input to be passed to standard input and returns the combined output (stdout and stderr) as a slice of bytes and an error.
// The command is run in a temporary directory that is removed at the end. This temporary directory contains all files given in input
//
// If the command was run successfully with no error, err is null.
// If the command was run successfully with a status code != 0, the error is a generic "failed". Note that stderr is included in the output
// If there was an error in running the command (command not found, ...) then the error is a generic "internal error". The underlying error is logged.
func runWithStdin(ctx context.Context, cmd ShellCommand) (output string, err error) {
	dir, err := ioutil.TempDir("", "cheops.tmp.*")
	if err != nil {
		log.Printf("Couldn't create tmp dir: %v\n", err)
		return "", fmt.Errorf("internal error")
	}
	defer os.RemoveAll(dir)

	for filename, file := range cmd.Files {
		err := ioutil.WriteFile(path.Join(dir, filename), file, 0644)
		if err != nil {
			log.Printf("Couldn't write working file %s: %v\n", file, err)
			return "", fmt.Errorf("internal error")
		}
	}

	execCommand := exec.CommandContext(ctx, "sh")
	execCommand.Dir = dir
	log.Printf("Running sh on %v\n", cmd.Command)

	stdin, err := execCommand.StdinPipe()
	if err != nil {
		log.Printf("Couldn't get stdin: %v\n", err)
		return "", fmt.Errorf("internal error")
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, cmd.Command)
	}()

	out, err := execCommand.CombinedOutput()
	if err != nil {
		log.Printf("Couldn't run [%s]: %v\n", cmd.Command, err)
		return "", fmt.Errorf("internal error")
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		log.Printf("[%s]: %s\n", cmd.Command, scanner.Text())
	}

	if execCommand.ProcessState != nil && !execCommand.ProcessState.Success() {
		return "", fmt.Errorf("failed")
	}

	return string(out), nil
}

func Handle(ctx context.Context, commands []ShellCommand) (replies []string, err error) {
	replies = make([]string, 0)
	doRun := true

	for i, cmd := range commands {
		var output string
		if doRun {
			out, err2 := runWithStdin(ctx, cmd)
			if err2 != nil {
				err = err2
				doRun = false
				log.Printf("Error running command %d, skipping %d\n", i+1, len(commands)-i-1)
			}
			output = out
		}
		replies = append(replies, output)
	}
	return replies, err
}
