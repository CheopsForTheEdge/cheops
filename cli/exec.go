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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"cheops.com/model"
	"github.com/alecthomas/kong"
	"golang.org/x/sync/errgroup"
)

var sitesRE = regexp.MustCompile("[^&,]+")
var cmdWithFilesRE = regexp.MustCompile("{([^}]+)}")

type ExecCmd struct {
	Command    string `help:"Command to run" required:"" short:""`
	Sites      string `help:"sites to deploy to" required:""`
	LocalLogic string `help:"Local logic file" required:""`
	Config     string `help:"config file" required:""`
}

func (e *ExecCmd) Run(ctx *kong.Context) error {
	for _, file := range []string{e.Config, e.LocalLogic} {
		fi, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("Invalid config or local logic file %s: %v\n", file, err)
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("Invalid config or local logic file %s: file mode is %v\n", file, fi.Mode())
		}
	}

	var b bytes.Buffer
	seenFiles := make(map[string]struct{})
	mw := multipart.NewWriter(&b)
	err := mw.WriteField("sites", e.Sites)
	if err != nil {
		log.Fatalf("Error with sites: %v\n", err)
	}
	writeFile(mw, "local-logic", e.LocalLogic)
	seenFiles["local-logic"] = struct{}{}
	writeFile(mw, "config.json", e.Config)
	seenFiles["config.json"] = struct{}{}

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
		log.Fatalf("Error with command: %v\n", err)
	}

	err = mw.Close()
	if err != nil {
		log.Fatalf("Error with form: %v\n", err)
	}

	// Determine if it is nosync or not
	// This will change the endpoint and the sites to run to
	f, err := os.Open(e.Config)
	if err != nil {
		log.Fatalf("Error opening config: %v\n", err)
	}
	defer f.Close()
	var config model.ResourceConfig
	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		log.Fatalf("Error opening config: %v\n", err)
	}
	if config.Mode == model.ModeNosync {
		hosts := sitesRE.FindAllString(e.Sites, -1)
		var g errgroup.Group
		for _, host := range hosts {
			host := host // necessary for goroutines
			g.Go(func() error {
				url := fmt.Sprintf("http://%s:8079/direct", host)
				return doRequest(url, mw.Boundary(), b)
			})
		}
		return g.Wait()
	} else {
		host := sitesRE.FindString(e.Sites)
		if host == "" {
			return fmt.Errorf("No host to send request to")
		}
		url := fmt.Sprintf("http://%s:8079", host)
		return doRequest(url, mw.Boundary(), b)
	}
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

	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("Couldn't run request on %s: %v\n", url, res.Status)
	}

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
		fmt.Printf("%s %s\n", r.Status, r.Site)
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
