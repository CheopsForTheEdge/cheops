# Tutorial 02 : Deploy federated consul clusters (g5k)

Some steps are automated with Enos but there are still lots of manips by hand for now (in the future those could be automated as well).

## Setup two kubernetes clusters with Enos

On a g5k frontend :

### First install only

```bash
virtualenv -p python3 venv 
source venv/bin/activate
pip install -U pip
pip install enoslib
```

If you don't already have a ~/.python-grid5000.yaml file with verify_ssl: False (or anything better)
```bash
cat > ~/.python-grid5000.yaml <<EOF 
verify_ssl: False
EOF

chmod 600 ~/.python-grid5000.yaml
```

### Deploy first cluster

Copy the enos-consul.py file on the frontend.
By default we make a reservation on site Rennes and nodes parasilo. Launch with :

```bash
python enos-consul.py
```

### Deploy second cluster

Just change the job name in enos-consul.py to anything else. Re-run the python command to deploy the second cluster.

Now we have two fonctionnal k8s clusters. Lets deploy Consul on each cluster with federation enabled. 

## Deploy federated consul clusters

### Primary cluster

To achieve federation, we need to chose one cluster as the primary. On the master node of that cluster, copy the primary-consul-values.yaml file (available in the assets above) and run it with the command :

```bash
helm install -f primary-consul-values.yaml hashicorp hashicorp/consul
```

Then copy the proxydefault.yaml file and apply it :

```bash
kubectl apply -f proxydefault.yaml
```

Now you have to get the secret generated for the federation :

```bash
kubectl get secret consul-federation -o yaml > consul-federation-secret.yaml
```

### Joining clusters

For all other clusters (on the master node), copy the secret you got from the primary in a file named consul-federation-secret.yaml and apply it : 

```bash
kubectl apply -f consul-federation-secret.yaml
```

Then copy the consul-values.yaml file (in the assets above) and run it with the command :

```bash
helm install -f consul-values.yaml hashicorp hashicorp/consul
```

## Check everything is running 

Open a terminal on the master node of any one of your clusters. Run the following command : 

```bash
kubectl exec -it consul-server-0 -- /bin/sh
```

Now check if the federation is working properly : 

```bash
consul members -wan
```
You should see all your clusters with the status "Alive". This means remote discovery is working.
Now lets try to list the services provided on your local cluster :

```bash
curl -k https://127.0.0.1:8501/v1/catalog/services
```

You should see the consul service and the mesh gateway service.

Now let's see the services available on the remote cluster :

```bash
curl -k https://127.0.0.1/v1/catalog/services?dc=YOUR_DC_NAME
```

Replace YOUR_DC_NAME with the name of a remote cluster. 

## Deploy Google boutique

See tutorial 01 section *Test with Google boutique*, and do it for all your clusters.




