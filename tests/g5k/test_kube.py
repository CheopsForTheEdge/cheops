#!/usr/bin/env python

# This test
# - creates a resource
# - lets it be synchronized
# - blocks synchronization
# - updates the resource on 2 different nodes
# - restores synchronization and lets it run
#
# After that we should have the same resource everywhere

import random
import string
import json
import yaml
import enoslib as en

import tests
import firewall_block
import g5k

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

class TestKube(tests.CheopsTest):

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
                'sites': (None, g5k.sites),
                'type': (None, 'create'),
                'config': (None, json.dumps(config)),
                'create_recipe.yml': ('create_recipe.yml', yaml.dump(recipe)),
            })
            self.wait_and_verify(id)

            recipe['spec']['replicas'] = 2
            self.do(id, 0, {
                'command': (None, f"sudo kubectl apply -f apply_recipe.yml"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
                'apply_recipe.yml': ('apply_recipe.yml', yaml.dump(recipe)),
            })
            patch = "spec:\n  replicas: 3"
            self.do(id, 1, {
                'command': (None, f"sudo kubectl patch deployment deployment-{id} -p '${patch}'"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
            })
            recipe['spec']['replicas'] = 4
            self.do(id, 2, {
                'command': (None, f"sudo kubectl replace -f replace_recipe.yml"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
                'replace_recipe.yml': ('replace_recipe.yml', yaml.dump(recipe)),
            })
            self.wait_and_verify(id)

            # Check that the spec is the same everywhere. Other fields may differ but they don't matter
            self.verify_shell(f"sudo kubectl get deployment deployment-{id} -o json | jq '.spec.replica'")

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
                'sites': (None, g5k.sites),
                'type': (None, 'create'),
                'config': (None, json.dumps(config)),
                'create_recipe.yml': ('create_recipe.yml', yaml.dump(recipe)),
            })
            self.wait_and_verify(id)

            firewall_block.activate([g5k.roles_for_hosts[2]])

            recipe['spec']['replicas'] = 2
            self.do(id, 0, {
                'command': (None, f"sudo kubectl apply -f apply_recipe.yml"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
                'apply_recipe.yml': ('apply_recipe.yml', yaml.dump(recipe)),
            })
            patch = "spec:\n  replicas: 3"
            self.do(id, 1, {
                'command': (None, f"sudo kubectl patch deployment deployment-{id} -p '${patch}'"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
            })
            recipe['spec']['replicas'] = 4
            self.do(id, 2, {
                'command': (None, f"sudo kubectl replace -f replace_recipe.yml"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
                'replace_recipe.yml': ('replace_recipe.yml', yaml.dump(recipe)),
            })

            firewall_block.deactivate([g5k.roles_for_hosts[2]])
            self.wait_and_verify(id)

            # Check that the spec is the same everywhere. Other fields may differ but they don't matter
            self.verify_shell(f"sudo kubectl get deployment deployment-{id} -o json | jq '.spec.replica'")

    def test_simple_with_failure(self):
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
                'sites': (None, g5k.sites),
                'type': (None, 'create'),
                'config': (None, json.dumps(config)),
                'create_recipe.yml': ('create_recipe.yml', yaml.dump(recipe)),
            })
            self.wait_and_verify(id)

            firewall_block.activate([g5k.roles_for_hosts[2]])

            recipe['spec']['replicas'] = 2
            self.do(id, 0, {
                'command': (None, f"sudo kubectl apply -f apply_recipe.yml"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
                'apply_recipe.yml': ('apply_recipe.yml', yaml.dump(recipe)),
            })
            patch = "spec:\n  replicas: 3"
            self.do(id, 1, {
                'command': (None, f"sudo kubectl patch deployment deployment-{id} -p '${patch}'"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
            })
            recipe['spec']['replicas'] = "not-a-number"
            self.do(id, 2, {
                'command': (None, f"sudo kubectl replace -f replace_recipe.yml"),
                'sites': (None, g5k.sites),
                'type': (None, 'apply'),
                'replace_recipe.yml': ('replace_recipe.yml', yaml.dump(recipe)),
            })
            print(f"Operation for {id} on {g5k.hosts[2]} is expected to have failed")

            firewall_block.deactivate([g5k.roles_for_hosts[2]])
            firewall_block.wait(g5k.hosts)
            # Don't verify here, the last one is supposed to fail

            # Check that the spec is the same everywhere. Other fields may differ but they don't matter
            self.verify_shell(f"sudo kubectl get deployment deployment-{id} -o json | jq '.spec.replica'")

if __name__ == '__main__':
    g5k.init()
    firewall_block.deactivate(g5k.roles_for_hosts)

    import unittest
    unittest.main()
