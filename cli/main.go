package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
)

var (
	id           = flag.String("id", "", "Id of resource, must not be empty")
	locationsRaw = flag.String("locations", "", "Locations of resource")
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s: %s -id <id> -locations <locations> -- <cmd>\n", os.Args[0], os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(flag.CommandLine.Output(), "  cmd\n\tthe command to execute for this resource\n")
	os.Exit(1)
}

func parseArgs() (cmd string, locations []string) {
	flag.Usage = usage
	flag.Parse()

	if *id == "" {
		usage()
	}

	var startCommand int
	for i, content := range os.Args {
		if content == "--" {
			startCommand = i + 1
			break
		}
	}
	if startCommand == 0 {
		usage()
	}

	cmd = strings.Join(os.Args[startCommand:], " ")

	if *locationsRaw != "" {
		locations = strings.Split(*locationsRaw, ",")
	}
	return
}

func main() {
	cmd, locations := parseArgs()

	// TODO cache id -> host to reuse it
	var host string
	if len(locations) > 0 {
		host = locations[rand.Intn(len(locations))]
	}

	if host == "" {
		log.Fatal("No host to send request to")
	}

	url := fmt.Sprintf("http://%s:8079/%s", host, *id)
	req, err := http.NewRequest("POST", url, strings.NewReader(cmd))
	if err != nil {
		log.Fatalf("Error building request: %v\n", err)
	}
	req.Header.Set("X-Cheops-Location", strings.Join(locations, ","))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Couldn't run request: %v\n", err)
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
	if err := sc.Err(); err != nil {
		fmt.Println(err)
	}
}
