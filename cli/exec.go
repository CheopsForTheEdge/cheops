// exec.go allows executing a command on a given resource and given locations
//
// Usage:
// $ cli --command "kubectl create deployment {deployment.yml}" --sites "S1&S2" --local-logic ll.cue --config config.json
//
// In the command, any file wrapped with {} will be sent in the request (so that it can be used remotely).
// The local-logic and config file must exist; they will also be sent.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
)

var sitesRE = regexp.MustCompile("[^&,]+")
var cmdWithFilesRE = regexp.MustCompile("{([^}]+)}")

type ExecCmd struct {
	Command    string `help:"Command to run" required:"" short:""`
	Sites      string `help:"sites to deploy to" required:""`
	Id         string `help:"id of the resource" required:""`
	LocalLogic string `help:"Local logic file"`
	Config     string `help:"config file"`
}

func (e *ExecCmd) Run(ctx *kong.Context) error {
	var b bytes.Buffer
	seenFiles := make(map[string]struct{})
	mw := multipart.NewWriter(&b)

	// Write sites
	err := mw.WriteField("sites", e.Sites)
	if err != nil {
		return fmt.Errorf("Error with sites: %v\n", err)
	}

	// Write resource files
	_, err = os.Stat(e.Config)
	if err == nil {
		writeFile(mw, "config.json", e.Config)
		seenFiles["local-logic"] = struct{}{}
	}
	_, err = os.Stat(e.LocalLogic)
	if err == nil {
		writeFile(mw, "local-logic", e.LocalLogic)
		seenFiles["local-logic"] = struct{}{}
	}

	// Command management
	// We replace every named file that will be local (such as {/etc/hostname}) with a base file
	// ({hostname} in this example), and we add the file to the request form.
	// We also add a suffix .i if the same name appears multiple times
	replacements := make([][]string, 0)
	matches := cmdWithFilesRE.FindAllStringSubmatch(e.Command, -1)
	for _, match := range matches {
		path := match[1]
		_, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("Invalid referenced file %s: %v\n", path, err)
		}
		basename := filepath.Base(path)
		name := basename
		i := 0
		for {
			if _, seen := seenFiles[name]; !seen {
				break
			}
			i += 1
			name = fmt.Sprintf("%s.%d", basename, i)
		}
		writeFileOrDir(mw, name, path)
		replacements = append(replacements, []string{path, name})
	}

	command := e.Command
	for _, replacement := range replacements {
		command = strings.Replace(command, replacement[0], replacement[1], 1)
	}
	err = mw.WriteField("command", command)
	if err != nil {
		return fmt.Errorf("Error with command: %v\n", err)
	}

	err = mw.Close()
	if err != nil {
		return fmt.Errorf("Error with form: %v\n", err)
	}

	host := sitesRE.FindString(e.Sites)
	if host == "" {
		return fmt.Errorf("No host to send request to")
	}
	if e.Id != url.PathEscape(e.Id) {
		return fmt.Errorf("id has url-unsafe characters, please choose something else")
	}
	uu := fmt.Sprintf("http://%s:8079/%s", host, e.Id)
	u, err := url.Parse(uu)
	if err != nil {
		return fmt.Errorf("Invalid parameters for host or id: %v\n", err)
	}
	return doRequest(u.String(), mw.Boundary(), b)

}

func doRequest(url, boundary string, body bytes.Buffer) error {
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return fmt.Errorf("Error building request for %s: %v\n", url, err)
	}
	req.Header.Set("Content-Length", strconv.Itoa(body.Len()))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Couldn't run request on %s: %v\n", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Couldn't run request on %s: %v\n", url, res.Status)
	}

	type reply struct {
		Site   string
		Status string
		Output string
	}
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		var r reply
		err := json.Unmarshal(sc.Bytes(), &r)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%s %s\t%s\n", r.Status, r.Site, r.Output)
	}
	return sc.Err()

}

func writeFileOrDir(mw *multipart.Writer, filename, path string) {
	stat, err := os.Stat(path)
	if err != nil {
		log.Fatalf("Error with %s: %v\n", filename, err)
	}
	if stat.Mode().IsRegular() {
		writeFile(mw, filename, path)
	} else {
		writeDir(mw, filename, path)
	}
}

func writeFile(mw *multipart.Writer, filename, path string) {
	fw, err := mw.CreateFormFile(filename, filename)
	if err != nil {
		log.Fatalf("Error with %s: %v\n", filename, err)
	}
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error with %s: %v\n", filename, err)
	}
	fi, err := f.Stat()
	if err != nil {
		log.Fatalf("Error with %s: %v\n", filename, err)
	}
	if !fi.Mode().IsRegular() {
		log.Fatalf("Error with %s: %v\n", filename, err)
	}

	defer f.Close()
	_, err = io.Copy(fw, f)
	if err != nil {
		log.Fatalf("Error with %s: %v\n", filename, err)
	}
}

func writeDir(mw *multipart.Writer, filename, path string) {
	var b bytes.Buffer
	mixedw := multipart.NewWriter(&b)
	defer mixedw.Close()

	err := filepath.WalkDir(path, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("zip: cannot add non-regular file")
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`attachment; name="%s"`, filename))
		h.Set("Content-Type", "application/octetstream")
		ffw, err := mixedw.CreatePart(h)
		if err != nil {
			log.Fatalf("Error with %s: %v\n", filename, err)
		}
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(ffw, f)
		return err
	})
	if err != nil {
		log.Fatalf("Error with %s: %v\n", path, err)
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, filename))
	h.Set("Content-Type", "multipart/mixed")
	h.Set("Boundary", mixedw.Boundary())
	fw, err := mw.CreatePart(h)
	if err != nil {
		log.Fatalf("Error with %s: %v\n", filename, err)
	}

	_, err = io.Copy(fw, &b)
	if err != nil {
		log.Fatalf("Error with %s: %v\n", path, err)
	}

}
