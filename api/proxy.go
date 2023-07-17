package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"cheops.com/env"
)

func proxyWaitBeforeWritingReply(ctx context.Context, host string, w http.ResponseWriter, method string, path string, header http.Header, body []byte) (*http.Response, error) {

	u, err := url.Parse(fmt.Sprintf("http://%s/%s", host, path))
	if err != nil {
		return nil, err
	}

	newreq, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))

	if err != nil {
		http.Error(w, "can't build proxy request", http.StatusInternalServerError)
		return nil, err
	}

	for key, vals := range header {
		for _, val := range vals {
			newreq.Header.Add(key, val)
		}
	}
	newreq.Header.Add("X-Forwarded-For", env.Myip)

	resp, err := http.DefaultClient.Do(newreq)
	if err != nil {
		http.Error(w, "can't send to backend", http.StatusInternalServerError)
		log.Println(err)

		// Not a blocking error
		return nil, nil
	}

	log.Printf(`-> [%s] %s %s`, host, newreq.Method, newreq.URL.String())
	log.Printf("Request headers: \n")
	for key, vals := range newreq.Header {
		log.Printf("\t%s: %v\n", key, vals)
	}

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

	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, bytes.NewReader(respbuf))
	if err != nil {
		http.Error(w, "can't write reply", http.StatusInternalServerError)
		log.Println(err)

		// Not a blocking error
		return nil
	}

	log.Printf(`<- [%s] %d %s`, host, resp.StatusCode, resp.Request.URL.String())
	log.Printf("Response headers: \n")
	for key, vals := range resp.Header {
		log.Printf("\t%s: %v\n", key, vals)
	}

	return nil

}

func proxy(ctx context.Context, host string, w http.ResponseWriter, method string, path string, header http.Header, body []byte) error {
	resp, err := proxyWaitBeforeWritingReply(ctx, host, w, method, path, header, body)
	if err != nil {
		return err
	}
	return proxyWriteReply(resp, w, host)
}
