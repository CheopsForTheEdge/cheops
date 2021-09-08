module cheops.com/broker

go 1.16

replace cheops.com/openstack => ../glue/openstack

require (
	cheops.com/openstack v0.0.0-00010101000000-000000000000
	github.com/rabbitmq/amqp091-go v0.0.0-20210823000215-c428a6150891
)
