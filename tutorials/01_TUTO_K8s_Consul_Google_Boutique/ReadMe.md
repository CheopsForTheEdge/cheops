# TUTORIAL 01 : Deploy Google Boutique on Consul & K8s (G5k)

## Grid5000

Make a reservation for 5 nodes (in the following we will work with nodes from Rennes) : 

``` oarsub -I -l host=5,walltime=4 -p "cluster='parasilo'" -t deploy```

Deploy a debian image on all your nodes :

``` kadeploy3 -f $OAR_NODE_FILE -e debian10-x64-base -k ```

## Start a Kubernetes cluster

To deploy a kubernetes cluster on your nodes, use the .sh files provided :

- Master node : run the initMaster.sh file on the node you want as Master (keep this terminal open)
- Worker nodes : run the initWorker.sh file on all the remaining nodes

After completion, the master node will yield an output similar to : 

``` kubeadm join 172.16.97.1:6443 --token ars98w.h4gr6g7d6y54l6ky \ --discovery-token-ca-cert-hash [TRUNCATED]```

**Copy the command you got on your master node** and run it on each worker node. Wait a little. On your master node, run the following command :

``` kubectl get nodes ```

You should see all nodes as 'Ready'. If not wait and retry.

## Deploy Consul with k8s

### Install Consul

Now that you have a running k8s cluster, let's deploy Consul : 
- Install helm with the following commands (on the master node) : 

``` curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 ```

``` chmod 700 get_helm.sh ```

``` ./get_helm.sh  ```

- Now add the official HashiCorp Consul Helm chart repo :

```helm repo add hashicorp https://helm.releases.hashicorp.com```

You should see something like this :

*"hashicorp" has been added to your repositories*

Now create a yaml config file to override some values from the original chart : 

```yaml
 cat > consul-values.yaml <<EOF
global:
  domain: consul
  datacenter: dc1

server:
  replicas: 1
  bootstrapExpect: 1

client:
  enabled: true
  grpc: true

ui:
  enabled: true

connectInject:
  enabled: true
  default: true

controller:
  enabled: true
EOF 
```

- Create a persistent volume so that the consul server can do its thing :
    - first we will create a directory :

    ``` mkdir /consul ```

    ``` mkdir /consul/data ```

    - then make the yaml file for the creation of the PV :

    ```yaml
    cat > /consul/data/pv.yaml <<EOF 
    apiVersion: v1
    kind: PersistentVolume
    metadata:
        name: data-default-hashicorp-consul-server-0
        labels:
            type: local
    spec:
        capacity:
            storage: 10Gi
        accessModes:
            - ReadWriteOnce
        hostPath:
            path: "/consul/data"
    EOF
    ```

- Run that .yaml to create a PV :

``` kubectl apply -f /consul/data/pv.yaml ```

- use helm to deploy Consul using the consul-values.yaml file from above : 

``` helm install -f consul-values.yaml hashicorp hashicorp/consul ```

- check you installation :

``` watch kubectl get pods ```

This command will let you observe the creation of all the consul services. Wait a few minutes. If the hashicorp-consul-server keeps crashing, then see the following troubleshooting section (Ctrl-C to exit the watch). 

### Troubleshooting

Your consul-server needs to have permissions for the /consul/data/ directory. Locate on which node the Consul server is running:

``` kubectl get pods --output=wide ```

Now just ssh to the corresponding node and allow all permissions to that directory : 

``` chmod -R 777 /consul ```

Go back to the master node and wait until all the pods are "Running".

```
NAME                                                              READY   STATUS    RESTARTS   AGE
hashicorp-consul-c4k5v                                            1/1     Running   0          8m7s
hashicorp-consul-connect-injector-webhook-deployment-5d758bb6ch   1/1     Running   0          8m9s
hashicorp-consul-controller-6b8746f57-9825h                       1/1     Running   0          8m9s
hashicorp-consul-g9x7g                                            1/1     Running   0          8m8s
hashicorp-consul-hrw2l                                            1/1     Running   0          8m8s
hashicorp-consul-l7hdt                                            1/1     Running   0          8m9s
hashicorp-consul-server-0                                         1/1     Running   4          8m8s
hashicorp-consul-webhook-cert-manager-f65f7c6fd-f8wx4             1/1     Running   0          8m9s
```

### Access Consul UI with your web browser

This can be done with a simple SSH tunnel. First, figure out the corresponding IP :

``` kubectl get services ```

In this example we have 10.105.80.30 and port 80.

``` 
hashicorp-consul-ui   ClusterIP   10.105.80.30   <none>   80/TCP 
```
Now on your local machine :

``` ssh -L 8080:<the IP from previous step>:80 <your-master-node (ex : parasilo-3)>.rennes.g5k -l root ```

Once access is configured, Consul UI will be available at http://localhost:8080



## Test with Google Boutique

### Deployment

1 - Clone the repository

``` git clone https://github.com/GoogleCloudPlatform/microservices-demo.git ```

2 - Remove original config

``` rm microservices-demo/release/kubernetes-manifests.yaml ```

3 - Replace with custom kubernetes-manifests.yaml available above

4 - Deploy the application

``` kubectl apply -f microservices-demo/release/kubernetes-manifests.yaml ```

5 - Wait until all pods are Ready

``` watch kubectl get pods ```

You should have something similar to this :

```
NAME                                                              READY   STATUS    RESTARTS   AGE
adservice-65b6b95d5d-vsz5m                                        3/3     Running   0          3m25s
cartservice-78bbfb4889-2qwt7                                      3/3     Running   1          3m26s
checkoutservice-599877775b-t5c2k                                  3/3     Running   0          3m26s
currencyservice-84b6788f86-hrwlq                                  3/3     Running   0          3m25s
emailservice-7f96b8856-vpft9                                      3/3     Running   0          3m26s
frontend-c5fd8bc45-gj6h8                                          3/3     Running   0          3m26s
hashicorp-consul-c4k5v                                            1/1     Running   0          62m
hashicorp-consul-connect-injector-webhook-deployment-5d758bb6ch   1/1     Running   0          62m
hashicorp-consul-controller-6b8746f57-9825h                       1/1     Running   0          62m
hashicorp-consul-g9x7g                                            1/1     Running   0          62m
hashicorp-consul-hrw2l                                            1/1     Running   0          62m
hashicorp-consul-l7hdt                                            1/1     Running   0          62m
hashicorp-consul-server-0                                         1/1     Running   4          62m
hashicorp-consul-webhook-cert-manager-f65f7c6fd-f8wx4             1/1     Running   0          62m
loadgenerator-598c44d5cc-hv5pj                                    3/3     Running   3          3m25s
paymentservice-79f65fd884-nqmk2                                   3/3     Running   0          3m26s
productcatalogservice-567d764f56-76jzx                            3/3     Running   0          3m25s
recommendationservice-7bb9fb6545-428c5                            3/3     Running   0          3m26s
redis-cart-79c48644f-fh8cv                                        3/3     Running   0          3m25s
shippingservice-586fdf64d8-ngr6j                                  3/3     Running   0          3m25s
```

You can also check the new services with Consul UI at http://localhost:8080

### Interact with the application frontend on your web browser

This is exactly the same as before with Consul UI :

1 - Figure out the corresponding IP (this time we want the frontend service)

``` kubectl get services ```

```
frontend   ClusterIP   10.99.129.196   <none>   80/TCP
```

2 - Forward accordingly

```ssh -L 18500:<the IP from previous step>:80 <your-master-node>.rennes.g5k -l root ```

3 - Access at http://localhost:18500 




