package main

import (
        "log"
        "cheops.com/openstack"
//        "math/rand"
         amqp "github.com/rabbitmq/amqp091-go"
	 "encoding/json"
	 "io/ioutil"
)


func failOnError(err error, msg string) {
        if err != nil {
                log.Fatalf("%s: %s", msg, err)
        }
}



func main() {
        conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
        failOnError(err, "Failed to connect to RabbitMQ")
        defer conn.Close()

        ch, err := conn.Channel()
        failOnError(err, "Failed to open a channel")
        defer ch.Close()

        q, err := ch.QueueDeclare(
                "rpc_queue", // name
                false,       // durable
                false,       // delete when unused
                false,       // exclusive
                false,       // no-wait
                nil,         // arguments
        )
        failOnError(err, "Failed to declare a queue")

        err = ch.Qos(
                1,     // prefetch count
                0,     // prefetch size
                false, // global
        )
        failOnError(err, "Failed to set QoS")

        msgs, err := ch.Consume(
                q.Name, // queue
                "",     // consumer
                false,  // auto-ack
                false,  // exclusive
                false,  // no-local
                false,  // no-wait
                nil,    // args
        )
        failOnError(err, "Failed to register a consumer")
//        log.Println(msgs.body)
        forever := make(chan bool)

        go func() {
                for d := range msgs {
			var response string
			log.Println(d.Body)
			temp := int(d.Body[0])
			if temp == 48 {
                        	response = openstack.Get()
                        	log.Printf("%s", response)
			} else{
				var result map[string]interface{}
			        json.Unmarshal(d.Body, &result)

			        log.Println(result)
				jsonString, err := json.Marshal(result)
				if err != nil {
     				       log.Println(err)
    				}
				_ = ioutil.WriteFile("deployment.json", jsonString, 0755)
			       // log.Println(byteValue)
				response = openstack.Deploy()
				log.Printf("%s", response)


			}
                        err = ch.Publish(
                                "",        // exchange
                                d.ReplyTo, // routing key
                                false,     // mandatory
                                false,     // immediate
                                amqp.Publishing{
                                        ContentType:   "text/plain",
                                        CorrelationId: d.CorrelationId,
                                        Body:          []byte(response),
                                })
                        failOnError(err, "Failed to publish a message")

                        d.Ack(false)
                }
        }()

        log.Printf(" [*] Awaiting RPC requests")
        <-forever
}
