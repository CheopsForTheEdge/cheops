module cheops.com

go 1.16

replace (
	cheops.com/api => ./api
	cheops.com/backends => ./backends
	cheops.com/client => ./client
	cheops.com/config => ./config
	cheops.com/database => ./database
	cheops.com/env => ./env
	cheops.com/k8s => ./glue/k8s/
	cheops.com/openstack => ./glue/openstack
	cheops.com/operation => ./operation
)

require (
	github.com/alecthomas/kong v0.8.1
	github.com/gorilla/mux v1.8.0
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.8.0 // indirect
)
