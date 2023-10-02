package api

import (
	"context"
	"net/http"

	"cheops.com/backends"
)

func proxy(ctx context.Context, host string, w http.ResponseWriter, method string, path string, header http.Header, body []byte) error {
	_, _, err := backends.HandleKubernetes(ctx, method, path, header, body)
	return err
}
