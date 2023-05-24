module cheops.com

go 1.16

replace (
	cheops.com/api => ./api
	cheops.com/client => ./client
	cheops.com/config => ./config
	cheops.com/database => ./database
	cheops.com/k8s => ./glue/k8s/
	cheops.com/openstack => ./glue/openstack
	cheops.com/operation => ./operation
)

require (
	github.com/arangodb/go-driver v0.0.0-20210825071748-9f1169c6a7dc
	github.com/gorilla/mux v1.8.0
	github.com/rabbitmq/amqp091-go v1.3.4
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/viper v1.11.0
	github.com/tcnksm/go-httpstat v0.2.0
)
