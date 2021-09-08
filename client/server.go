package main
import (
    "fmt"
    "log"
    "io/ioutil"
    "os"
    "net/http"
    "io"
    "math/rand"
    amqp "github.com/rabbitmq/amqp091-go"
)



func randomString(l int) string {
        bytes := make([]byte, l)
        for i := 0; i < l; i++ {
                bytes[i] = byte(randInt(65, 90))
        }
        return string(bytes)


}



func randInt(min int, max int) int {
        return min + rand.Intn(max-min)
}


func Broker_Client(url string, deploy []byte ) string {
        conn, err := amqp.Dial(url)
        failOnError(err, "Failed to connect to RabbitMQ")
        defer conn.Close()

        ch, err := conn.Channel()
        failOnError(err, "Failed to open a channel")
        defer ch.Close()

        q, err := ch.QueueDeclare(
                "",    // name
                false, // durable
                false, // delete when unused
                true,  // exclusive
                false, // noWait
                nil,   // arguments
        )
        failOnError(err, "Failed to declare a queue")

        msgs, err := ch.Consume(
                q.Name, // queue
                "",     // consumer
                true,   // auto-ack
                false,  // exclusive
                false,  // no-local
                false,  // no-wait
                nil,    // args
        )
        failOnError(err, "Failed to register a consumer")

        corrId := randomString(32)

        err = ch.Publish(
                "",          // exchange
                "rpc_queue", // routing key
                false,       // mandatory
                false,       // immediate
                amqp.Publishing{
                        ContentType:   "text/plain",
                        CorrelationId: corrId,
                        ReplyTo:       q.Name,
                        Body:          deploy,
                })
        failOnError(err, "Failed to publish a message")
        var res string
        for d := range msgs {
                if corrId == d.CorrelationId {
                        res = string(d.Body)
                        failOnError(err, "Failed to convert body to integer")
                        break
                }
        }
        log.Printf("\n\n\n&s",res)
        return res
}



func failOnError(err error, msg string) {
        if err != nil {
                log.Fatalf("%s: %s", msg, err)
        }
}


func deployHandler(w http.ResponseWriter, r *http.Request) {
	jsonFile, err := os.Open("deployment.json")
	if err != nil {
            fmt.Println(err)
    }
	byteValue, _ := ioutil.ReadAll(jsonFile)
	res1 := Broker_Client("amqp://guest:guest@172.16.192.21:5672/", byteValue)
        log.Printf("%s", res1)


	io.WriteString(w, res1)
}


func getHandler(w http.ResponseWriter, r *http.Request) {
        //fmt.Fprintf(w, "Hello!")
        res1 := Broker_Client("amqp://guest:guest@172.16.192.21:5672/", []byte("0"))
        log.Printf("%s", res1)

        res2 := Broker_Client("amqp://guest:guest@172.16.192.21:5672/", []byte("0"))
        log.Printf("%s", res2)

        Final_res := res1 + res2
        io.WriteString(w, Final_res)
    }

func main() {
//	Cluster1 := "amqp://guest:guest@172.16.192.9:5672/"
//	Cluster2 := "amqp://guest:guest@172.16.192.9:5672/"
//	Cluster3 := "amqp://guest:guest@172.16.192.9:5672/"
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/deploy",deployHandler)

        fmt.Printf("Starting server at port 8080\n")
        if err := http.ListenAndServe(":8080", nil); err != nil {
       		log.Fatal(err)
    	}
}
