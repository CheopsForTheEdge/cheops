package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"cheops.com/replicator"
	"github.com/alecthomas/kong"
)

type ShowCmd struct {
	Id   string `help:"Id of resource, must not be empty" required:""`
	Hint string `help:"One location where the resource is"`
}

func (s *ShowCmd) Run(ctx *kong.Context) error {
	hosts, err := getAndShow(s.Hint, s.Id)
	if err != nil {
		return err
	}

	for _, host := range hosts {
		if host != s.Hint {
			_, err := getAndShow(host, s.Id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getAndShow(host, id string) (allHosts []string, err error) {
	url := fmt.Sprintf("http://%s:5984/cheops/%s", host, id)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var m replicator.ResourceDocument
	err = json.NewDecoder(res.Body).Decode(&m)
	if err != nil {
		return nil, err
	}

	bytes, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s\n", bytes)

	return m.Locations, nil
}
