package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/alecthomas/kong"
)

type ShowCmd struct {
	Id   string `help:"Id of resource, must not be empty" required:""`
	Hint string `help:"One location where the resource is"`
}

func (s *ShowCmd) Run(ctx *kong.Context) error {
	url := fmt.Sprintf("http://%s:5984/cheops/%s", s.Hint, s.Id)
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = io.Copy(os.Stdout, res.Body)
	return err
}
