// Package api offers the API entrypoints for the different packages
package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
)

func BestEffort(port int) {

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.EscapedPath()

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		log.Printf(`* %s %s`, method, r.URL.String())

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
			if isAlreadyForwarded(r, sites) {
				continue
			}

			w.Header().Add("Trailer", header)

			g.Go(func() error {
				e := &emptyResponseWriter{
					// by default, assume other sites are unreachable
					statusCode: http.StatusInternalServerError,
				}

				err := proxy(ctx, site, e, method, path, r.Header, body)
				statuses[header] = e.statusCode
				return err
			})
		}

		resp, err := proxyWaitBeforeWritingReply(r.Context(), "127.0.0.1:8283", w, method, path, r.Header, body)
		if err != nil {
			log.Println(err)
			return
		}

		err = g.Wait()
		if err != nil {
			log.Println(err)
			// Not blocking, we don't return yet
		}

		for header, code := range statuses {
			w.Header().Set(header, fmt.Sprintf("%d", code))
		}
		proxyWriteReply(resp, w, "127.0.0.1:8283")

	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
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
