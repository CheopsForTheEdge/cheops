module cheops.com

go 1.16


replace (
	cheops.com/openstack => ./glue/openstack
	cheops.com/database => ./database
	cheops.com/operation => ./operation
	cheops.com/api => ./api
)

require (
	github.com/arangodb/go-driver v0.0.0-20210825071748-9f1169c6a7dc
	github.com/gorilla/mux v1.8.0
	github.com/justinas/alice v1.2.0
	github.com/segmentio/ksuid v1.0.4
	github.com/rabbitmq/amqp091-go v0.0.0-20210823000215-c428a6150891 // indirect
)
