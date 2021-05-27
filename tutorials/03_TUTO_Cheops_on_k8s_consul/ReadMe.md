# Install & test Cheops on a kubernetes/consul environment 

## Deploy a kubernetes cluster with consul 

Follow the readme on this [gitlab repository](https://gitlab.inria.fr/aszymane/enos-consul/-/tree/master) 

## Deploy test services

We have been testing cheops with two custom services (their code is available on this repository under cheops/test).
To deploy them easily, ssh to the machine where your k8s master lives and execute these commands :


For the serviceA :
```bash
kubectl run app-a --annotations consul.hashicorp.com/connect-service-upstreams=app-b:1234 --image=mariedonnie/servicea --port=5001
```

For the serviceB :
```bash
kubectl run app-b --image=mariedonnie/serviceb --port=5002
```

Check that the pods were correctly created with the command :

```bash
kubectl get pods
```
You should see something like this (mind the 3/3, this means we have three containers running inside the pod, one for the app itself, one for the envoy sidecar proxy and one for the consul agent) :

TBD => INSERT IMAGE OF RESULT (g5k not working properly ATM)

Try to test the two services with the instructions in cheops/test/serviceA (resp. serviceB). 
IMPORTANT : you will have to change the IP address used in the instructions for the one k8s assigned. To find it, use :

```bash
kubectl get pods -o wide
```
Locate the pod you want (either app-a or app-b) and look at the corresponding IP. This will be the one you will want to use. For example, if your IP is 10.244.2.3, then the curl instruction will be :

curl -X GET http://**10.244.2.3**:5001/resourcea/1


## Deploy Cheops

You can either use the current image of cheops we have on dockerhub :

```bash
kubectl run cheops --image=juzdzewski/juzdzew:latest --port=8080
```

Or you can make your own image if you wish to change the code (cheops is still a work in progress and is not 100% functional). Here is one possible way to do so :

```bash
git clone https://gitlab.inria.fr/discovery/cheops.git
cd cheops/cheops
```

From there you have access to all the classes and you can change the code (cheops is written in go). We provide a dockerfile located in cheops/cheops. When you are ready to build, go to that directory and build the image with :

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
```

Check that everything went as intended :

```bash
kubectl get pods -o wide
```

Just like for the services, you should see cheops being ready and 3/3. You can quickly test to curl the root (the IP you need to use is the one k8s assigned to cheops and provided by the above command):

```bash
curl YOUR_IP:8080
```

The terminal should reply with the message *"Welcome home"*.

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

Now apply this config entry :

```bash
kubectl apply -f proxydefaults.yaml
```

We are now ready to define a service router entry. For now, we will ask the Envoy around app-b to re-route incoming traffic towards Cheops. Several points before we go forward : 

- the routing to cheops is what we want to do but is not working properly at the moment. However if you want to get a grasp on routing with a working example, you can try to deploy another *serviceB* (name it something like app-c), modify the resource of ID 1 (see instructions on cheops/serviceB for this), then re-route traffic from app-b to app-c (instead of cheops) and test it with the curl from app-a :  curl -X POST http://IP_OF_APP_A:5001/resourceafromb/localhost:1234

- the application running inside the cluster must follow a micro-service architecture and be fully integrated with the service mesh (meaning all upstreams must be explicitly declared so that Envoy can be used)

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
EOF
```

Apply that config file :

```bash
kubectl apply -f routingB.yaml
```

What we have done so far is saying that when, from serviceA (or any service with an upstream to app-b) we need to query app-b, the Envoy around app-b will forward the request to cheops, which should in turn re-direct the request appropriately. This is the part where we are now : having cheops interpret the request and forwarding it appropriately.

We can however make sure that the Envoy re-routing works. Open a shell on the app-a :

```bash
kubectl exec -it app-a -- /bin/sh
```

We are now inside the app-a container (by default because don't forget there are 3 containers inside this pod). You can try this query :

```bash
curl localhost:1234
```

Now localhost:1234 refers to the app-b root (by the upstream rule), so without routing we should get the message *"service b"*. But as we have routed the request to cheops root, we are getting the message *"Welcome home"* which is the cheops answer when hitting its own root.
So routing works.  







