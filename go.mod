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
	github.com/goombaio/dag v0.0.0-20181006234417-a8874b1f72ff
	github.com/gorilla/mux v1.8.0
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
)
