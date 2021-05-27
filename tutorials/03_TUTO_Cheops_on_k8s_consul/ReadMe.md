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

Try to test the two services with the instructions in cheops/test/serviceA. 

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
kubectl get pods
```

Just like for the services, you should see cheops being ready and 3/3. You can quickly test to curl the root :

```bash
curl 





