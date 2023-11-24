// exec.go allows executing a command on a given resource and given locations
//
// Usage:
// $ cli --id <resource-id> --locations "site1,site2" mkdir /tmp/foo

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/alecthomas/kong"
)

type ExecCmd struct {
	Id        string   `help:"Id of resource, must not be empty" required:""`
	Locations []string `help:"Locations of resource" optional:""`
	Command   string   `arg:""`
}

func (e *ExecCmd) Run(ctx *kong.Context) error {
	// TODO cache id -> host to reuse it
	var host string
	if len(e.Locations) > 0 {
		host = e.Locations[rand.Intn(len(e.Locations))]
	}

	if host == "" {
		return fmt.Errorf("No host to send request to")
	}

	url := fmt.Sprintf("http://%s:8079/%s", host, e.Id)
	req, err := http.NewRequest("POST", url, strings.NewReader(e.Command))
	if err != nil {
		return fmt.Errorf("Error building request: %v\n", err)
	}
	req.Header.Set("X-Cheops-Location", strings.Join(e.Locations, ","))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Couldn't run request: %v\n", err)
	}
	defer res.Body.Close()

	type reply struct {
		Site   string
		Status string
	}
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		var r reply
		err := json.Unmarshal(sc.Bytes(), &r)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%s\t%s\n", r.Site, r.Status)
	}
	return sc.Err()
}
