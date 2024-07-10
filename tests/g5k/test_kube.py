#!/usr/bin/env python

# This test
# - creates a resource
# - lets it be synchronized
# - blocks synchronization
# - updates the resource on 2 different nodes
# - restores synchronization and lets it run
#
# After that we should have the same resource everywhere

import sys
import os
import enoslib as en
import unittest
import random, string
import json
import requests
import socket
import enoslib as en
import yaml
import tempfile

import firewall_block

# Hack
if any(['g5k-jupyterlab' in path for path in sys.path]):
    print("Running on Grid'5000 notebooks, applying workaround for https://intranet.grid5000.fr/bugzilla/show_bug.cgi?id=13606")
    print("Before:", sys.path)
    sys.path.insert(1, os.environ['HOME'] + '/.local/lib/python3.9/site-packages')
    print("After:", sys.path)


# Make it work on nantes and grenoble
hostname = socket.gethostname()
if hostname == "fnantes":
    site = "nantes"
    cluster = "econome"
elif hostname == "fgrenoble":
    site = "grenoble"
    cluster = "dahu"
else:
    site = "nantes"
    cluster = "econome"

# Get the cluster
en.init_logging()
network = en.G5kNetworkConf(type="prod", roles=["my_network"], site=site)
conf = (
    en.G5kConf.from_settings(job_type=[], walltime="01:50:00", job_name="cheops")
    .add_network_conf(network)
    .add_machine(
        roles=["cheops"],
        cluster=cluster,
        nodes=4,
        primary_network=network,
    )
    .finalize()
)
provider = en.G5k(conf)
rroles, networks = provider.init()
en.sync_info(rroles, networks)

roles = rroles["cheops"]
hosts = [r.alias for r in roles]
sites = '&'.join(hosts[:3])
roles_for_hosts = [role for role in roles if role.alias in hosts[:3]]

# Ensure firewall allows sync
firewall_block.deactivate(roles_for_hosts)

recipe = yaml.safe_load("""
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
  generation: 1
  labels:
    app: kubernetes-bootcamp
  name: kubernetes-bootcamp
  namespace: default
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: kubernetes-bootcamp
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: kubernetes-bootcamp
    spec:
      containers:
      - image: gcr.io/google-samples/kubernetes-bootcamp:v1
        imagePullPolicy: IfNotPresent
        name: kubernetes-bootcamp
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
""")

class TestKube(unittest.TestCase):
    def do(self, id, index, request):
        r1 = requests.post(f"http://{hosts[index]}:8079/{id}", files=request)
        self.assertEqual(200, r1.status_code, id)

    def wait_and_verify(self, id):
        firewall_block.wait(hosts)

        replies = [requests.get(f"http://{host}:5984/cheops/{id}") for host in hosts[:3]]
        for reply in replies:
            self.assertEqual(200, reply.status_code)
            self.assertEqual(replies[0].json(), reply.json())
        contents = [reply.json() for reply in replies]
        for content in contents:
            self.assertEqual(content['Operations'], contents[0]['Operations'])

        # Make sure the replies are all ok
        for host in hosts[:3]:
            query = {"selector": {
                "Type": "REPLY",
                "Site": host,
                "ResourceId": id
            }}
            r = requests.post(f"http://{host}:5984/cheops/_find", json=query, headers={"Content-Type": "application/json"})
            for doc in r.json()['docs']:
                self.assertEqual("OK", doc['Status'], f"status is KO {doc}")

    def verify_kube(self, command):
        # Make sure the directory has the correct content everywhere
        with en.actions(roles=roles_for_hosts) as p:
            p.shell(command)
            results = p.results

        contents = [content.payload['stdout'] for content in results.filter(task="shell")]
        for content in contents[1:]:
            self.assertEqual(contents[0], content)

    def test_simple(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            recipe['metadata']['name'] = f"deployment-{id}"
            config = {'RelationshipMatrix': [
                {'Before': 'create', 'After': 'apply', 'Result': 'take-both-reverse-order'},
                {'Before': 'apply', 'After': 'create', 'Result': 'take-both-keep-order'},
                {'Before': 'apply', 'After': 'apply', 'Result': 'take-one'},
                {'Before': 'create', 'After': 'patch', 'Result': 'take-both-keep-order'},
                {'Before': 'patch', 'After': 'create', 'Result': 'take-both-reverse-order'},
            ]}

            self.do(id, 0, {
                'command': (None, f"sudo kubectl create -f create_recipe.yml"),
                'sites': (None, sites),
                'type': (None, 'create'),
                'config': (None, json.dumps(config)),
                'create_recipe.yml': ('create_recipe.yml', yaml.dump(recipe)),
            })
            self.wait_and_verify(id)

            recipe['spec']['replicas'] = 2
            self.do(id, 0, {
                'command': (None, f"sudo kubectl replace -f apply_recipe.yml"),
                'sites': (None, sites),
                'type': (None, 'apply'),
                'apply_recipe.yml': ('apply_recipe.yml', yaml.dump(recipe)),
            })
            patch = "spec:\n  replicas: 3"
            self.do(id, 1, {
                'command': (None, f"sudo kubectl patch deployment deployment-{id} -p '${patch}'"),
                'sites': (None, sites),
                'type': (None, 'patch'),
            })
            recipe['spec']['replicas'] = 4
            self.do(id, 2, {
                'command': (None, f"sudo kubectl replace -f replace_recipe.yml"),
                'sites': (None, sites),
                'type': (None, 'apply'),
                'replace_recipe.yml': ('replace_recipe.yml', yaml.dump(recipe)),
            })
            self.wait_and_verify(id)

            # Check that the spec is the same everywhere. Other fields may differ but they don't matter
            self.verify_kube(f"sudo kubectl get deployment deployment-{id} -o json | jq '.spec'")

    def test_simple_with_disconnect(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            recipe['metadata']['name'] = f"deployment-{id}"
            config = {'RelationshipMatrix': [
                {'Before': 'create', 'After': 'apply', 'Result': 'take-both-reverse-order'},
                {'Before': 'apply', 'After': 'create', 'Result': 'take-both-keep-order'},
                {'Before': 'apply', 'After': 'apply', 'Result': 'take-one'},
                {'Before': 'create', 'After': 'patch', 'Result': 'take-both-keep-order'},
                {'Before': 'patch', 'After': 'create', 'Result': 'take-both-reverse-order'},
                {'Before': 'apply', 'After': 'patch', 'Result': 'take-both-keep-order'},
                {'Before': 'patch', 'After': 'apply', 'Result': 'take-both-reverse-order'},
            ]}

            self.do(id, 0, {
                'command': (None, f"sudo kubectl create -f create_recipe.yml"),
                'sites': (None, sites),
                'type': (None, 'create'),
                'config': (None, json.dumps(config)),
                'create_recipe.yml': ('create_recipe.yml', yaml.dump(recipe)),
            })
            self.wait_and_verify(id)

            firewall_block.activate([roles_for_hosts[2]])

            recipe['spec']['replicas'] = 2
            self.do(id, 0, {
                'command': (None, f"sudo kubectl replace -f apply_recipe.yml"),
                'sites': (None, sites),
                'type': (None, 'apply'),
                'apply_recipe.yml': ('apply_recipe.yml', yaml.dump(recipe)),
            })
            patch = "spec:\n  replicas: 3"
            self.do(id, 1, {
                'command': (None, f"sudo kubectl patch deployment deployment-{id} -p '${patch}'"),
                'sites': (None, sites),
                'type': (None, 'patch'),
            })
            recipe['spec']['replicas'] = 4
            self.do(id, 2, {
                'command': (None, f"sudo kubectl replace -f replace_recipe.yml"),
                'sites': (None, sites),
                'type': (None, 'apply'),
                'replace_recipe.yml': ('replace_recipe.yml', yaml.dump(recipe)),
            })

            firewall_block.deactivate([roles_for_hosts[2]])
            self.wait_and_verify(id)

            # Check that the spec is the same everywhere. Other fields may differ but they don't matter
            self.verify_kube(f"sudo kubectl get deployment deployment-{id} -o json | jq '.spec'")


if __name__ == '__main__':
    unittest.main()
