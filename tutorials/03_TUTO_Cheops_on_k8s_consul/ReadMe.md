# Install & test Cheops on a kubernetes/consul environment 

## Deploy a kubernetes cluster with consul 

Follow the readme on this [gitlab repository](https://gitlab.inria.fr/aszymane/enos-consul/-/tree/master) 

## Deploy test services

**BREAKING CHANGE : Consul released a new version in which they introduce the transparent proxy feature (among other things). This feature is enabled by default and might break our setup. Further testing is required to decide if we want to disable it or not.**

We have been testing cheops with two custom services (their code is available on this repository under cheops/test).
To deploy them easily, ssh to the machine where your k8s master lives and execute these commands :

*Note that as of consul-k8s v0.26.0, we are now required to expose our service as a Kubernetes service in order to use the service mesh.* 

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
You should see something like this (mind the 3/3, this means we have three containers running inside the pod, one for the app itself, one for the envoy sidecar proxy and one for the consul agent) :

TBD => INSERT IMAGE OF RESULT (g5k not working properly ATM)

Try to test the two services with the instructions in cheops/test/serviceA (resp. serviceB). 

**IMPORTANT :** you will have to change the IP address used in the instructions for the one k8s assigned. To find it, use :

```bash
kubectl get pods -o wide
```
Locate the pod you want (either app-a or app-b) and look at the corresponding IP. This will be the one you will want to use. For example, if your IP is 10.244.2.3, then the curl instruction will be :

curl -X GET http://**10.244.2.3**:5001/resourcea/1


## Cheops installation tutorial

Cheops is composed of two complementary services : the **core** handles most of the operations and the **connector** handles remote queries to and from distant sites. We will have to deploy each of these independently.

### Deploy the Cheops core

#### Use provided docker image

You can either use the current image of cheops we have on dockerhub :

```bash
kubectl run cheops --image=juzdzewski/juzdzew:latest --port=8080
kubectl expose pod cheops --port=8080 --name=cheops
```

#### Create your own image

Or you can make your own image if you wish to change the code (cheops is still a work in progress and is not 100% functional). Here is one possible way to do so :

```bash
git clone --single-branch --branch Matthieu https://gitlab.inria.fr/discovery/cheops.git
cd cheops/cheops
```

From there you have access to all the classes and you can change the code (cheops is written in *Go*). We provide a dockerfile located in cheops/cheops. When you are ready to build, go to that directory and build the image with :

```bash
docker build -t yourAccountName/yourRepoName:latest .
docker login
```

You will be prompted to enter your docker credentials. After successfully login in :

```bash
docker push yourAccountName/yourRepoName:latest
```

You can now deploy your own image with :

```bash
kubectl run cheops --image=yourAccountName/yourRepoName:latest --port=8080
kubectl expose pod cheops --port=8080 --name=cheops
```

Check that everything went as intended :

```bash
kubectl get pods -o wide
```

#### Configure and test core install

**### TEST ###**

Just like for the services, you should see cheops being ready and 3/3. You can quickly test to curl the root (the IP you need to use is the one k8s assigned to cheops and provided by the above command):

```bash
curl YOUR_IP:8080
```

The terminal should reply with the message *"Welcome home"*.

**### Configure ###**

Cheops needs to know the IP address of the *hashicorp-consul-server-0* pod. For now, the procedure is as follows :

**1. Get the IP :**

  ```bash
  kubectl get pods -o wide
  ```
  Locate **hashicorp-consul-server-0** and copy the IP address (CONSUL_IP).
  Locate **cheops** and remember the IP address (CHEOPS_IP).
  In the following, replace CHEOPS_IP and CONSUL_IP with your own values. 

**2. Configure Cheops :**

We will now feed this IP to Cheops so that it can use it to interact with Consul later on. 

```bash
curl -X POST CHEOPS_IP:8080/consulIP -d '{"ip":"CONSUL_IP"}' -H "Content-Type: application/json"
```

### Deploy the Cheops connector

**TO BE DONE**

## Setup service routing

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

We are now ready to define a service router entry. For now, we will ask the Envoy around app-b to re-route incoming traffic towards Cheops. Several points before we go forward : 

- if you have followed this tutorial, then your Cheops connector is not deployed yet (coming soon) and so you can't properly test forwarding.

- the application running inside the cluster must follow a micro-service architecture and be fully integrated with the service mesh (meaning all upstreams must be explicitly declared so that Envoy can be used) **<== this condition might change if we decide to use the new Consul features.**

Now we are going to define our service router as such :

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

What we have done so far is saying that when, from serviceA (or any service with an upstream to app-b) we need to query app-b, the Envoy around app-b will forward the request to cheops on the endpoint **/Appb** (the */o* part is a workaround : Envoy would append the original prefix path to /Appb and we don't want that, so we use a dummy path */o* so that Envoy can add anything to it and in Go we juste use regex to ignore that part), which should in turn re-direct the request appropriately.

As the Cheops connector is not yet integrated, you can only test forwarding locally : 

```bash
kubectl exec -it app-a -- /bin/sh
```

We are now inside the app-a container (by default because don't forget there are 3 containers inside this pod). You can try this query :

```bash
curl localhost:1234
```

Remember that localhost:1234 refers to the app-b root (by the upstream rule), so what will happen is the Envoy around app-b will intercept the request and forward it to Cheops on the endpoint */Appb/o*. Then Cheops will redirect the request to app-b and the result should be : "service b".
So routing works. In the future, we will introduce the Cheops connector and the ability to forward to distant sites using our DSL scope-lang. 







