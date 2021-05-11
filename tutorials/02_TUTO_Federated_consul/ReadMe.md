# Tutorial 02 : Deploy federated consul clusters (g5k)

Some steps are automated with Enos but there are still lots of manips by hand for now (in the future those could be automated as well).

## Setup two kubernetes clusters with Enos

On a g5k frontend :

### First install only

First, git clone the project. Then setup the basics :

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

Move to the working directory : 

```bash
cd cheops/tutorials/02_TUTO_Federated_consul/
```

### Deploy first cluster

By default we make a reservation on site Rennes and nodes parasilo. Launch with :

```bash
python enos-consul.py
```

**If this is your first launch, you might get stuck with this message from 10 to 15min, just wait :**

```
The Vagrant executable cannot be found. Please check if it is in the system path.
Note: Openstack clients not installed
Unverified HTTPS request is being made. Make sure to do this on purpose or set verify_ssl in the configuration file
```

### Deploy second cluster

Just change the job name in enos-consul.py to anything else. Re-run the python command to deploy the second cluster.

Now we have two fonctionnal k8s clusters. Lets deploy Consul on each cluster with federation enabled. 

## Deploy federated consul clusters

### Primary cluster

To achieve federation, we need to chose one cluster as the primary. Connect as **root** on the master node of that cluster. To find the master node, juste look at which node is listed under this task (here it is ecotype-44) : 

```
TASK [enoslib_adhoc_command] *******************************************************************************************
 [started TASK: enoslib_adhoc_command on ecotype-44.nantes.grid5000.fr]
changed: [ecotype-44.nantes.grid5000.fr]
```

Once you are connected as root on that node, apply the primary-consul-values.yaml with the command :

```bash
helm install -f primary-consul-values.yaml hashicorp hashicorp/consul
```

You should see some warnings, don't mind them and wait until you have this confirmation message :

```
NAME: hashicorp
LAST DEPLOYED: Fri May  7 22:23:14 2021
NAMESPACE: default
STATUS: deployed
REVISION: 1
NOTES:
Thank you for installing HashiCorp Consul!

Now that you have deployed Consul, you should look over the docs on using
Consul with Kubernetes available here:

https://www.consul.io/docs/platform/k8s/index.html


Your release is named hashicorp.

To learn more about the release, run:

  $ helm status hashicorp
  $ helm get all hashicorp
  ```
Then apply the proxydefault.yaml file :

```bash
kubectl apply -f proxydefault.yaml
```

Now you have to get the secret generated for the federation :

```bash
kubectl get secret consul-federation -o yaml > consul-federation-secret.yaml
```

### Joining clusters

For all other secondary clusters (on their respective master node), copy the secret you just got from the primary in a file named consul-federation-secret.yaml and apply it : 

```bash
kubectl apply -f consul-federation-secret.yaml
```

Then apply the consul-values.yaml file with the command :

```bash
helm install -f consul-values.yaml hashicorp hashicorp/consul
```

Finally, apply the proxydefault.yaml file :

```bash
kubectl apply -f proxydefault.yaml
```

## Check everything is running 

Wait a few seconds in order to let the remote discovery feature to finish. 
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
curl -k https://127.0.0.1:8501/v1/catalog/services?dc=YOUR_DC_NAME
```

Replace YOUR_DC_NAME with the name of a remote cluster. 

## Deploy Google boutique

See tutorial 01 section *Test with Google boutique*, and do it for all your clusters.




