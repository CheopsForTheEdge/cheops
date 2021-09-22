package client

import (
	"fmt"
	"log"
	"io/ioutil"
	"os"
	"net/http"
	"io"
	"math/rand"
	amqp "github.com/rabbitmq/amqp091-go"
	// "github.com/gorilla/mux"
	"cheops.com/operation"
	"cheops.com/request"
	"cheops.com/endpoint"
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
	res1 := Broker_Client("amqp://guest:guest@10.44.61.255:5672/", byteValue)
        log.Printf("%s", res1)


	io.WriteString(w, res1)
}


func getHandler(w http.ResponseWriter, r *http.Request) {
        //fmt.Fprintf(w, "Hello!")
        res1 := Broker_Client("amqp://guest:guest@10.44.61.255:5672/", []byte("0"))
        log.Printf("%s", res1)

        res2 := Broker_Client("amqp://guest:guest@10.44.61.255:5672/", []byte("0"))
        log.Printf("%s", res2)

        Final_res := res1 + res2
        io.WriteString(w, Final_res)
}


func routing() {
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/deploy",deployHandler)
	http.HandleFunc("/", homeLink)
	// Replication
	http.HandleFunc("/replication", operation.CreateLeaderFromOperationAPI).Methods("POST")
	http.HandleFunc("/replicant/{metaID}", operation.GetReplicant).Methods("GET")
	http.HandleFunc("/replicant/{metaID}", operation.AddReplica).Methods("PUT")
	http.HandleFunc("/replicant/{metaID}", operation.DeleteReplicant).Methods("DELETE")
	http.Handle("/replicants", operation.GetAllReplicants).Methods("GET")
	// Endpoint
	http.HandleFunc("/endpoint/getaddress/{Site}", endpoint.GetAddressAPI).Methods("GET")
	// Database
	// Operation
	http.HandleFunc("/operation", operation.CreateOperationAPI).Methods("POST")
	http.HandleFunc("/operation/execute", operation.ExecuteOperationAPI).Methods("POST")
	// Broker - Driver
	http.HandleFunc("/scope",request.ExtractScope).Methods("GET")
	http.HandleFunc("/scope/forward",request.RedirectRequest).Methods("POST")
	http.HandleFunc("/Appb/{flexible:.*}", request.Appb).Methods("GET")
	http.HandleFunc("/SendRemote", request.SendRemote).Methods("GET")
	http.HandleFunc("/RegisterRemoteSite", request.RegisterRemoteSite).Methods("POST")
	http.HandleFunc("/GetRemoteSite/{site}", request.GetRemoteSite).Methods("GET")

        fmt.Printf("Starting server at port 8080\n")
        if err := http.ListenAndServe(":8080", nil); err != nil {
       		log.Fatal(err)
    	}
}

// Default route
func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to Cheops!")
}
