package main

import (
        "log"
        "cheops.com/k8s"
//        "math/rand"
         amqp "github.com/rabbitmq/amqp091-go"
	 "encoding/json"
	 "bytes"
//	 "io/ioutil"
)


func failOnError(err error, msg string) {
        if err != nil {
                log.Fatalf("%s: %s", msg, err)
        }
}
/*type data struct{
	name string
	op string
	rs_name string
}*/

type Message map[string]interface{}
func deserialize(b []byte) (Message, error) {
    var msg Message
    buf := bytes.NewBuffer(b)
    decoder := json.NewDecoder(buf)
    err := decoder.Decode(&msg)
    return msg, err
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
			result, err := deserialize(d.Body)
			//result := data{}
			//var result map[string]interface{}
                        //json.Unmarshal(d.Body, &result)

                        log.Println(result)
			if result["operation"] == "check"{
				response = k8s.Cross_Check(result["resource_name"].(string))
			}else if result["operation"] == "createns"{
				response = k8s.Cross_Create(result["resource_name"].(string))
			}else if result["operation"] == "get"{
				response = k8s.Cross_Get(result["namespace"].(string),result["resource_name"].(string))
			}else if result["operation"] == "applycheck"{
				log.Println("\ncheck")
				response = k8s.Cross_App_Check(result["namespace"].(string),result["resource_name"].(string))
			}else if result["operation"] == "apply"{
				response = k8s.Cross_Apply(result["namespace"].(string),result["resource_name"].(string),result["depfile"].(string))
			}
			/*			temp := int(d.Body[0])
			if temp == 48 {
                        	response = k8s.Get()
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
				response = k8s.Deploy()
				log.Printf("%s", response)


			}
*/                        err = ch.Publish(
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
