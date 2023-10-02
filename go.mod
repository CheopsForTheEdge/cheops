module cheops.com

go 1.16

replace (
	cheops.com/api => ./api
	cheops.com/backends => ./backends
	cheops.com/client => ./client
	cheops.com/config => ./config
	cheops.com/database => ./database
	cheops.com/k8s => ./glue/k8s/
	cheops.com/openstack => ./glue/openstack
	cheops.com/operation => ./operation
	cheops.com/env => ./env
)

require (
	github.com/anacrolix/torrent v1.52.3
	github.com/arangodb/go-driver v0.0.0-20210825071748-9f1169c6a7dc
	github.com/gorilla/mux v1.8.0
	github.com/rabbitmq/amqp091-go v1.3.4
	github.com/rakoo/raft v0.0.0-20230616100538-e99ccd03fb74
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/viper v1.11.0
	golang.org/x/sync v0.1.0
	google.golang.org/grpc v1.46.2
	sigs.k8s.io/kustomize/kyaml v0.14.3
)
