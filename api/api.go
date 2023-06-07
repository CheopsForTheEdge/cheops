// Package api offers the API entrypoints for the different packages
package api

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	_ "cheops.com/kubernetes"
	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
)

func Routing() {

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf(`* %s %s`, r.Method, r.URL.String())
		for site := range sites {
			site := site
			host := strings.Split(site, ":")[0]
			header := fmt.Sprintf("X-status-%s", host)
			w.Header().Add("Trailer", header)
		}

		localRespChan := make(chan (*http.Response), 1)
		go func() {
			resp, err := proxyWaitBeforeWritingReply(r.Context(), "127.0.0.1:8283", w, r)
			if err != nil {
				log.Println(err)
			}
			localRespChan <- resp
		}()

		if isAlreadyForwarded(r, sites) {
			return
		}

		myip, ok := os.LookupEnv("MYIP")
		if !ok {
			log.Fatal("My IP must be given with the MYIP environment variable !")
		}
		log.Printf("my ip: %s\n", myip)
		r.Header.Add("X-Forwarded-For", myip)

		timeoutContext, timeoutCancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer timeoutCancel()
		g, ctx := errgroup.WithContext(timeoutContext)

		// status holds status by site
		// it's ok to use this because concurrent changes happen on different keys
		statuses := make(map[string]int)

		for site := range sites {
			site := site
			host := strings.Split(site, ":")[0]
			header := fmt.Sprintf("X-status-%s", host)
			g.Go(func() error {
				e := &emptyResponseWriter{
					// by default, assume other sites are unreachable
					statusCode: http.StatusInternalServerError,
				}
				err := proxy(ctx, site, e, r)
				statuses[header] = e.statusCode
				return err
			})
		}

		err := g.Wait()
		if err != nil {
			log.Println(err)
			// Not blocking, we don't return yet
		}

		localResp := <-localRespChan

		for header, code := range statuses {
			w.Header().Set(header, fmt.Sprintf("%d", code))
		}
		proxyWriteReply(localResp, w, "127.0.0.1:8283")
	})

	log.Fatal(http.ListenAndServe(":8080", router))
}

func isAlreadyForwarded(r *http.Request, sites map[string]struct{}) bool {
	// TODO: sites should be hostnames, so this can't always work
	for site := range sites {
		parts := strings.Split(site, ":")
		host := parts[0]
		for _, ff := range r.Header.Values("X-Forwarded-For") {
			if ff == host {
				return true
			}
		}
	}
	return false
}

// emptyResponseWriter stores the status code and discards everything else
type emptyResponseWriter struct {
	statusCode int
}

func (e *emptyResponseWriter) Header() http.Header {
	return http.Header(make(map[string][]string))
}
func (e *emptyResponseWriter) WriteHeader(statusCode int) {
	e.statusCode = statusCode
}
func (e *emptyResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func proxyWaitBeforeWritingReply(ctx context.Context, host string, w http.ResponseWriter, r *http.Request) (*http.Response, error) {
	defer r.Body.Close()
	reqbuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read request body", http.StatusInternalServerError)
		return nil, err
	}

	u := r.URL
	u.Host = host

	// Not filled by default
	u.Scheme = "http"

	newreq, err := http.NewRequestWithContext(ctx, r.Method, u.String(), bytes.NewReader(reqbuf))
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "can't build proxy request", http.StatusInternalServerError)
		return nil, err
	}

	for key, vals := range r.Header {
		for _, val := range vals {
			newreq.Header.Add(key, val)
		}
	}

	resp, err := http.DefaultClient.Do(newreq)
	if err != nil {
		http.Error(w, "can't send to backend", http.StatusInternalServerError)
		log.Println(err)

		// Not a blocking error
		return nil, nil
	}

	log.Printf(`->[%s] %s %s`, host, newreq.Method, newreq.URL.String())

	return resp, nil
}

func proxyWriteReply(resp *http.Response, w http.ResponseWriter, host string) error {
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
	w.WriteHeader(resp.StatusCode)

	log.Printf(`<- [%s] %d %s`, host, resp.StatusCode, resp.Request.URL.String())
	return nil

}

func proxy(ctx context.Context, host string, w http.ResponseWriter, r *http.Request) error {
	resp, err := proxyWaitBeforeWritingReply(ctx, host, w, r)
	if err != nil {
		return err
	}
	return proxyWriteReply(resp, w, host)
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
