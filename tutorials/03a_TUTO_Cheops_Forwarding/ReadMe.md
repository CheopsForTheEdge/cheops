# Setup Cheops remote forwarding

## Deploy two k8s clusters

To test forwarding on distant sites, we need at least two different clusters. Follow the readme on this [gitlab repository](https://gitlab.inria.fr/aszymane/enos-consul/-/tree/master). 

**IMPORTANT :** for the secondary cluster, don't forget to change the job's name in the file *enos-consul.py* (because of idempotence).

## Deploy test services A and B

For the serviceA :
```bash
kubectl run app-a --annotations consul.hashicorp.com/connect-service-upstreams=app-b:1234 --image=mariedonnie/servicea --port=5001
kubectl expose pod app-a --port=5001 --name=app-a
```

For the serviceB :
```bash
kubectl run app-b --image=mariedonnie/serviceb --port=5002
kubectl expose pod app-b --port=5002 --name=app-b
```

Check that the pods were correctly created with the command :

```bash
kubectl get pods
```
You should see *app-a* and *app-b* READY 2/2. If not, wait and retry the ``` kubectl get pods ``` command. 

## Install Cheops

Cheops is composed of two distinct components : the **core** and the **connector**. We will deploy them individually.

### Deploy and configure the core

The current solution requires us to manually feed the IP of consul server inside the code of cheops. So we will clone this project, modify the code, build and run our own image. First find the IP address of the pod *hashicorp-consul-server-0* with the ```kubectl get pods -o wide``` command.
Then clone the project with : 

```bash
git clone --single-branch --branch Matthieu https://gitlab.inria.fr/discovery/cheops.git
nano cheops/cheops/request
```
Locate the function **Appb** and replace the IP in the url field with your own. Ex :

```go
func Appb (w http.ResponseWriter, r *http.Request) {
        url := "http://{YOUR_IP}:8500/v1/catalog/service/app-b"
```

Save the change. We now have to build the image and push it on a public repo like DockerHub (make sure you have an account). 

```bash
docker build -t yourAccountName/yourRepoName:latest ./cheops/cheops
docker login
```

You will be prompted to enter your docker credentials. After successfully logging in, push the image as such :

```bash
docker push yourAccountName/yourRepoName:latest
```

Wait until the image is pushed. Then we can finally deploy the core and check all went well :

```bash
kubectl run cheops --image=yourAccountName/yourRepoName:latest --port=8080
kubectl expose pod cheops --port=8080 --name=cheops
kubectl get pods
```

### Deploy and configure the connector

First create a directory for the connector's code :

```bash
mkdir connector
cd connector
```

Then we will create a single main.go file :

```bash
cat > main.go <<EOF
package main

import (
        "fmt"
        "github.com/gorilla/mux"
        "io/ioutil"
        "log"
        "net/http"
)


func homeLink(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Welcome home!")
}



func HandleRemote(w http.ResponseWriter, r *http.Request) {
        url := "http://10.244.4.3:8080/" + r.Header.Get("service") + "/" + "o"
        req, _ := http.NewRequest("GET", url, nil)
        req.Header.Add("x-envoy-original-path", r.Header.Get("destPath"))
        client := &http.Client{}
        res, _ := client.Do(req)
        body, _ := ioutil.ReadAll(res.Body)
        w.Write(body)
}

func main() {
        router := mux.NewRouter().StrictSlash(true)
        router.HandleFunc("/", homeLink)
        router.HandleFunc("/HandleRemote", HandleRemote).Methods("GET")
        log.Fatal(http.ListenAndServe(":8081", router))
}
EOF
```

Once again, the current solution requires us to manually feed the IP of the core in this code. Use ```kubectl get pods -o wide``` and find the IP of the pod **cheops**. Then nano into the main.go file we just created and locate the HandleRemote function. As before, replace the IP part in the url field with this updated IP. Ex : 

```go
func HandleRemote(w http.ResponseWriter, r *http.Request) {
        url := "http://{CHEOPS_CORE_IP}:8080/" + r.Header.Get("service") + "/" + "o"
```

Save the change. We are ready to build, but first we need a dockerfile. Make sure your current directory is **connector** and then :

```bash
cat > dockerfile <<EOF
FROM golang:1.12.0-alpine3.9

RUN mkdir /app
RUN apk update && \
    apk upgrade && \
    apk add git

ADD . /app

RUN go get -d github.com/gorilla/mux

WORKDIR /app

RUN go build -o main .

EXPOSE 8081

CMD /app/main
EOF
```

Now we can finally build the image :

```bash
docker build -t yourAccountName/remote:latest .
```

We will deploy the connector as a traditional docker container instead of a kubernetes pod :

```bash
docker run -d -p 8081:8081 yourAccountName/remote:latest
```
Cheops is completely installed now.

## Consul configuration

According to the [consul documentation](https://www.consul.io/docs/connect/config-entries/service-router#interaction-with-other-config-entries), we need to declare that the protocol used by our services is HTTP-based. We will do this globally via the *proxy-defaults* config entry :

```bash
cat > proxydefaults.yaml <<EOF
apiVersion: consul.hashicorp.com/v1alpha1
kind: ProxyDefaults
metadata:
  name: global
spec:
  config:
    protocol: http
EOF
```

Apply this config entry :

```bash
kubectl apply -f proxydefaults.yaml
```

We are now ready to define a service router entry.

```bash
cat > routingB.yaml <<EOF
apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceRouter
metadata:
  name: app-b
spec:
  routes:
    - match:
        http:
          pathPrefix: /
      destination:
        service: cheops
        prefixRewrite: /Appb/o
EOF
```

Apply that config file :

```bash
kubectl apply -f routingB.yaml
```

We are done for now. **Repeat the entire process for the other cluster !**

## Final configuration

At this point we have two clusters with everything correctly installed and configured. There is one final configuration step : each cluster must be aware of its distant relative(s). First, we need to find the IP of the remote master node (use the method of your choice, you could for example just ```ping NAME_OF_DISTANT_NODE``` which will yield the IP). Then feed this IP into cheops :

```bash
curl -X POST http://LOCAL_CHEOPS_CORE_IP:8080/RegisterRemoteSite -d '{"remoteSiteName":"REMOTE_MASTER_IP"}' -H "Content-Type: application/json"
```
Repeat this process symetrically for the other site. **NOTE : make sure to remember the remoteSiteName you use as it will be necessary when writing the scope for remote forwarding** 

## Test forwarding

To better visualize if remote forwarding is functionnal, change the resource of ID=1 of the app-b pod in only one site (this way they are not identical anymore and we will be able to notice the difference). The readme in Cheops/Test/ServiceB in this git will provide information on how to do this. 

Let's start by testing local forwarding : 

Enter the pod app-a with ```kubectl exec -it app-a -- /bin/sh```. Then try :

```bash
curl localhost:1234/resourceb/1
```
The result should be the resource of ID 1 from the local app-b.

Now let's add the scope for remote forwarding. You will have to remember the exact name you used when registered the remote site.

```bash
curl localhost:1234/resourceb/1 -H {"Scope: app-b/NAME_OF_THE_DISTANT_SITE"}
```

The result should be the resource of ID 1 from the distant app-b. 








