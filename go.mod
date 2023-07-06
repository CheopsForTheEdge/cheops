module cheops.com

go 1.16

replace (
	cheops.com/api => ./api
	cheops.com/client => ./client
	cheops.com/config => ./config
	cheops.com/database => ./database
	cheops.com/k8s => ./glue/k8s/
	cheops.com/backends => ./backends
	cheops.com/openstack => ./glue/openstack
	cheops.com/operation => ./operation
)

require (
	github.com/arangodb/go-driver v0.0.0-20210825071748-9f1169c6a7dc
	github.com/gorilla/mux v1.8.0
	github.com/rabbitmq/amqp091-go v1.3.4
	github.com/rakoo/raft v0.0.0-20230616100538-e99ccd03fb74
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/viper v1.11.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.45.0
)
