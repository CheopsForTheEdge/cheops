// show.go displays a local view of a given resource from all the sites
// where it exists. The local view is from a site that is given in input.
//
// The given command will be executed on all sites and the output is returned.
// The special string __ID__ can be used in the command and will be replaced by
// the id that is given.
//
// Usage:
// $ cli show --id <resource-id> --from siteX --command <command>
//
// Output:
//
// {
//   "siteX": {
//     "Status": "OK", // can be KO or TIMEOUT
//     "Output": "..."
//   },
//   "siteY": {
//     "Status": "KO",
//     "Output": "..."
//   }
// }

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/alecthomas/kong"
)

type ShowCmd struct {
	Id      string `help:"Id of resource, must not be empty" required:""`
	From    string `help:"The site from which to query other sites" required:""`
	Command string `help:"Command to run" required:"" short:""`
}

func (s *ShowCmd) Run(ctx *kong.Context) error {
	u := fmt.Sprintf("http://%s:8079/show/%s", s.From, s.Id)

	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	mw.WriteField("command", s.Command)
	mw.Close()

	req, err := http.NewRequestWithContext(context.Background(), "POST", u, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Length", strconv.Itoa(b.Len()))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", mw.Boundary()))
	reply, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer reply.Body.Close()

	_, err = io.Copy(os.Stdout, reply.Body)
	return err
}
