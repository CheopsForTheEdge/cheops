package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"cheops.com/model"
	"github.com/alecthomas/kong"
)

type ShowCmd struct {
	Id   string `help:"Id of resource, must not be empty" required:""`
	Hint string `help:"One location where the resource is"`
}

func (s *ShowCmd) Run(ctx *kong.Context) error {
	byContent := make(map[string][]string)
	fetchedHosts := make(map[string]struct{})

	content, hosts, err := getContentAndOtherHosts(s.Hint, s.Id)
	if err != nil {
		return err
	}

	if byContent[content] == nil {
		byContent[content] = make([]string, 0)
	}
	byContent[content] = append(byContent[content], s.Hint)
	fetchedHosts[s.Hint] = struct{}{}

	hostsToFetch := make([]string, 0)
	for _, host := range hosts {
		hostsToFetch = append(hostsToFetch, host)
	}

	for {
		hasNewHosts := false
		for _, host := range hostsToFetch {
			hostsToFetch = hostsToFetch[1:]
			if _, ok := fetchedHosts[host]; !ok {
				hasNewHosts = true
				content, hosts, err := getContentAndOtherHosts(host, s.Id)
				if err != nil {
					return err
				}
				if byContent[content] == nil {
					byContent[content] = make([]string, 0)
				}
				byContent[content] = append(byContent[content], host)
				fetchedHosts[host] = struct{}{}

				for _, host := range hosts {
					hostsToFetch = append(hostsToFetch, host)
				}
			}
		}

		if !hasNewHosts {
			break
		}
	}

	for content, hosts := range byContent {
		for _, host := range hosts {
			fmt.Println(host)
		}
		indented := strings.ReplaceAll(content, "\n", "\n\t")
		indented = "\t" + indented
		fmt.Println(indented)
	}

	return nil
}

func getContentAndOtherHosts(host, id string) (content string, allHosts []string, err error) {
	url := fmt.Sprintf("http://%s:5984/cheops/%s", host, id)
	res, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	var m model.ResourceDocument
	err = json.NewDecoder(res.Body).Decode(&m)
	if err != nil {
		return "", nil, err
	}

	bytes, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		return "", nil, err
	}

	return string(bytes), m.Locations, nil
}
