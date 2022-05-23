module cheops.com/broker

go 1.16

replace cheops.com/openstack => ../glue/openstack

replace cheops.com/k8s => ../glue/k8s

require (
	cheops.com/k8s v0.0.0-00010101000000-000000000000
	github.com/rabbitmq/amqp091-go v0.0.0-20210823000215-c428a6150891
)
