package client

import (
	"cheops.com/config"
	"bytes"
	"cheops.com/endpoint"
	"cheops.com/operation"
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
)

var knownsites = config.Conf.Sites

var def_Cluster = []string{"amqp://guest:guest@172.16.192.10:5672/","amqp://guest:guest@172.16.192.11:5672/","amqp://guest:guest@172.16.192.13:5672/"}
var check_cluster = []string{"cluster1","cluster2","cluster3"}
func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)


}

type Message map[string]interface{}

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


func SendOperationToSites(w http.ResponseWriter, r *http.Request) {
	var op operation.Operation
	reqBody, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal([]byte(reqBody), &op)
	if err != nil {
		fmt.Fprintf(w, "There was an error reading the json: %s\n ", err)
		log.Fatal(err)
	}
	opByte, err := json.Marshal(op)
	for _, site := range op.Sites {
		address := endpoint.GetSiteAddress(site)
		result := Broker_Client(address, opByte)
		log.Printf("Result:%s\n", result)
		io.WriteString(w, result)
	}
}

func DeployHandler(w http.ResponseWriter, r *http.Request) {
	jsonFile, err := os.Open("deployment.json")
	if err != nil {
		fmt.Println(err)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	res1 := Broker_Client(def_Cluster[0], byteValue)
	log.Printf("%s", res1)


	io.WriteString(w, res1)
}


func GetHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "Hello!")
	res1 := Broker_Client(def_Cluster[0], []byte("0"))
	log.Printf("%s", res1)

	res2 := Broker_Client(def_Cluster[0], []byte("0"))
	log.Printf("%s", res2)

	Final_res := res1 + res2
	io.WriteString(w, Final_res)

}


func ReplicaHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Println(path)
	sPath := strings.Split(path, "/")
	log.Println(sPath)
	clusters := strings.Split(sPath[2],",")
	log.Println(sPath[3])
	jsonFile, err := os.Open("deployment.json")
	if err != nil {
		fmt.Println(err)
	}
	deploy_json, _ := ioutil.ReadAll(jsonFile)
	log.Println(deploy_json)
	res1 := " "
	for i := range(clusters){
		var res12 string
		log.Println(clusters[i])
		if clusters[i] == "cluster1"{
			res12 = Broker_Client(def_Cluster[0], deploy_json)
		}else if clusters[i] == "cluster2"{
			res12 = Broker_Client(def_Cluster[1], deploy_json)
		}else if clusters[i] == "cluster3"{
			res12 = Broker_Client(def_Cluster[2], deploy_json)
		}
		res1 = res1 + res12
	}
	log.Printf("%s", res1)
	io.WriteString(w, res1)

}
/*
func crossHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Println(path)
	sPath := strings.Split(path, "/")
	log.Println(sPath)
	clusters := strings.Split(sPath[2],",")
	log.Println(sPath[3])
        deploy_json := k8s.Get_Deploy(sPath[3])
	log.Println(deploy_json)
	res1 := " "
	for i := range(clusters){
		var res12 string
		log.Println(clusters[i])
		if clusters[i] == "cluster1"{
			res12 = Broker_Client(Cluster1, []byte(deploy_json))
		}else if clusters[i] == "cluster2"{
			res12 = Broker_Client(Cluster2, []byte(deploy_json))
		}else if clusters[i] == "cluster3"{
			res12 = Broker_Client(Cluster3, []byte(deploy_json))
		}
		res1 = res1 + res12
	}
        log.Printf("%s", res1)
	io.WriteString(w, res1)

}
*/

/*func serialize(msg1 Message) ([]byte, error) {
    var b bytes.Buffer
    encoder := json.NewEncoder(&b)
    err := encoder.Encode(msg1)
    return b.Bytes(), err
}*/


func Comm(clusters []string, content []byte) string{
	res1 := ""
	log.Println(strings.TrimSpace(strings.Join(clusters, "")))
	if strings.TrimSpace(strings.Join(clusters, "")) == "" {
		for i := range (def_Cluster){
			var res12 string
			res12 = Broker_Client(def_Cluster[i], []byte(content))
			log.Println(res1)
			res1 = res1 + "\n" + res12
		}
	}else{
		for i := range (clusters){
			var res12 string
			if clusters[i] == "cluster1"{
				res12 = Broker_Client(def_Cluster[0], []byte(content))
			}else if clusters[i] == "cluster2"{
				res12 = Broker_Client(def_Cluster[1], []byte(content))
			}else if clusters[i] == "cluster3"{
				res12 = Broker_Client(def_Cluster[2], []byte(content))
			}
			res1 = res1 + "\n" + res12
		}
	}
	return res1


}



func CrossHandler(w http.ResponseWriter, r *http.Request){

	path := r.URL.Path
	log.Println(path)
	sPath := strings.Split(path, "/")
	log.Println(sPath)
	msg := make(map[string]string)
	clusters := strings.Split(sPath[5],",")
	log.Println(clusters)
	log.Println(clusters, check_cluster)
	if sPath[2] == "create"{
		msg["operation"] = "check"
		msg["resource_name"] = sPath[4]
		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		err := encoder.Encode(msg)
		if err != nil{
			log.Fatalf("%s:%s","hi",err)
		}
		content1 := b.Bytes()
		//check_cluster := [2]string{"cluster1","cluster2"}
		res1 := Comm(check_cluster, content1)
		log.Println(res1)
		if strings.Contains(res1, "FALSE"){
			io.WriteString(w,"Namespace exist")
			return
		}
		msg["operation"] = "createns"
		//content = ns_name
	}else if sPath[3] == "apply"{
		msg["namespace"] = sPath[2]
		msg["operation"] = "applycheck"
		msg["resource_name"] = sPath[4]
		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		err := encoder.Encode(msg)
		if err != nil{
			log.Fatalf("%s:%s","hi",err)
		}
		content1 := b.Bytes()
		res1 := Comm(check_cluster, content1)
		log.Println(res1)
		if strings.Contains(res1, "FALSE"){
			io.WriteString(w,"Resource exist")
			return
		}
		//jsonFile, err := os.Open("deployment.json")
		dat, err := os.ReadFile("deployment.json")
		if err != nil {
			fmt.Println(err)
		}
		msg["depfile"] = string(dat)
		//json.Unmarshal(jsonFile, &msg["depfile"])
		msg["operation"] = "apply"

	}else{
		msg["namespace"] = sPath[2]
		msg["operation"] = sPath[3]
		msg["resource_name"] = sPath[4]
		//content_str :=
		//	content,err := json.Marshal([3]string{namespace,operation,resource_name})
		//if err != nil
		//        log.Fatalf("%s: %s", content, err)
		//}
		//content  := json.Marshal(content_str):
		//	log.Println(content)
		/*	var result map[string]interface{}
			json.Unmarshal(content, &result)
			log.Println(result)
		*/      //clusters := strings.Split(sPath[5],",")
		//log.Println(clusters)

		//deploy_json := k8s.Get_Deploy(sPath[3])
		//log.Println(deploy_json)
		/*content, err := serialize(msg)
			if err != nil {
		                log.Fatalf("%s: %s", content, err)
		        }*/
	}
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	err := encoder.Encode(msg)
	if err != nil{
		log.Fatalf("%s:%s","hi",err)
	}
	content := b.Bytes()
	//	}
	//res1 := ""
	//	clusters := strings.Split(sPath[5],",")
	//	log.Println(clusters)
	//	log.Println(reflect.TypeOf(content))
	/*if len (clusters) == 0{
			for i := range (def_Cluster){
				res1 := Broker_Client(def_Cluster[i], []byte(content))
				log.Println(res1)
			}
		}else{
			for i := range (clusters){
				var res12 string
				if clusters[i] == "cluster1"{
	                        res12 = Broker_Client(def_Cluster[0], []byte(content))
	               		}else if clusters[i] == "cluster2"{
	                        res12 = Broker_Client(def_Cluster[1], []byte(content))
	               	 	}else if clusters[i] == "cluster3"{
	                        res12 = Broker_Client(def_Cluster[2], []byte(content))
	                	}
	                res1 = res1 + res12
			}
		}*/
	log.Println("hello",clusters)
	res1 := Comm(clusters, content)
	log.Println(res1)
	io.WriteString(w,res1)

}

//func main() {
	//	Cluster1 := "amqp://guest:guest@172.16.192.9:5672/"
	//	Cluster2 := "amqp://guest:guest@172.16.192.9:5672/"
	//	Cluster3 := "amqp://guest:guest@172.16.192.9:5672/"
	//http.HandleFunc("/get", getHandler)
	//http.HandleFunc("/deploy",deployHandler)
	//http.HandleFunc("/cross/", crossHandler)
	//http.HandleFunc("/replica/", replicaHandler)
	//fmt.Printf("Starting server at port 8080\n")
	//if err := http.ListenAndServe(":8080", nil); err != nil {
	//	log.Fatal(err)
	//}
//}
