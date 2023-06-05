// Package api offers the API entrypoints for the different packages
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	_ "cheops.com/kubernetes"
	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
)

func Routing() {
	myip, ok := os.LookupEnv("MYIP")
	if !ok {
		log.Fatal("My IP must be given with the MYIP environment variable !")
	}

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index := 0
		g, ctx := errgroup.WithContext(r.Context())

		for site := range sites {
			site := site
			header := fmt.Sprintf("X-status-%s", site)
			if index == 0 {
				w.Header().Add("Trailer", header)
				g.Go(func() error {
					return proxy(ctx, site, w, r)
				})
			} else {
				if isAlreadyForwarded(r, sites) {
					continue
				}

				g.Go(func() error {
					e := emptyResponseWriter{}
					r.Header.Add("X-Forwarded-For", myip)
					err := proxy(ctx, site, e, r)
					w.Header().Set(header, strconv.Itoa(e.statusCode))
					return err

				})
			}
			index++
		}

		err := g.Wait()
		if err != nil {
			log.Println(err)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", router))
}

func isAlreadyForwarded(r *http.Request, sites map[string]struct{}) bool {
	for _, ff := range r.Header.Values("X-Forwarded-For") {
		if _, known := sites[ff]; known {
			return true
		}
	}
	return false
}

// emptyResponseWriter stores the status code and discards everything else
type emptyResponseWriter struct {
	statusCode int
}

func (e emptyResponseWriter) Header() http.Header {
	return http.Header(make(map[string][]string))
}
func (e emptyResponseWriter) WriteHeader(statusCode int) {
	e.statusCode = statusCode
}
func (e emptyResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func proxy(ctx context.Context, host string, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()
	reqbuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read request body", http.StatusInternalServerError)
		return err
	}

	u := r.URL

	parts := strings.Split(host, ":")
	if len(parts) == 2 {
		u.Host = host
	} else {
		u.Host = host + ":8283"
	}

	// Not filled by default
	u.Scheme = "http"

	newreq, err := http.NewRequestWithContext(ctx, r.Method, u.String(), bytes.NewReader(reqbuf))
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "can't build proxy request", http.StatusInternalServerError)
		return err
	}

	for key, vals := range r.Header {
		for _, val := range vals {
			newreq.Header.Add(key, val)
		}
	}

	timeout, _ := time.ParseDuration("3s")
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(newreq)
	if err != nil {
		http.Error(w, "can't send to backend", http.StatusInternalServerError)
		log.Println(err)

		// Not a blocking error
		return nil
	}
	defer resp.Body.Close()

	respbuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "can't read reply from backend", http.StatusInternalServerError)
		// Not a blocking error
		return nil
	}

	for key, vals := range resp.Header {
		for _, val := range vals {
			w.Header().Add(key, val)
		}
	}

	_, err = io.Copy(w, bytes.NewReader(respbuf))
	if err != nil {
		http.Error(w, "can't write reply", http.StatusInternalServerError)
		log.Println(err)

		// Not a blocking error
		return nil
	}

	indent := func(b []byte) string {
		var obj map[string]interface{}
		json.Unmarshal(b, &obj)
		indented, _ := json.MarshalIndent(obj, "", "\t")
		return string(indented)
	}
	headers := func(h http.Header) string {
		var asstring string
		for key, val := range h {
			asstring += fmt.Sprintf("%s=%s\n", key, val)
		}
		return asstring
	}

	log.Printf(`->[%s] %s %s
-> [%s] %s
<- [%s] %s
`, host, r.Method, r.URL.String(), host, indent(reqbuf), headers(resp.Header), host, indent(respbuf))
	return nil

}

// Checks if the request has data
func CheckRequestFilledHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		_, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Kindly enter data", r.Method, r.URL.String())

			fmt.Fprintf(w, "Kindly enter data")
			return
		} else {
			next.ServeHTTP(w, r)
		}
	}
	return http.HandlerFunc(fn)
}

// Default route
func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome home!")
}
