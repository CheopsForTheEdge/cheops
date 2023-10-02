package backends

import (
	"testing"

	jp "github.com/evanphx/json-patch"
)

func TestSitesFor(t *testing.T) {
	pod := `apiVersion: v1
kind: Pod
metadata:
  name: simpleapp-pod
  labels:
    app.kubernetes.io/name: SimpleApp
  annotations:
    locations: site1, site2, site3
spec:
  containers:
  - name: myapp-container
    image: busybox:1.28
    command: ['sh', '-c', 'echo The app is running! && sleep 3600']
  initContainers:
  - name: init-myservice
    image: busybox:1.28
    command: ['sh', '-c', "sleep 2"]
`
	sites, err := SitesFor("", "", nil, []byte(pod))
	if err != nil {
		t.Fatalf("Got error [%v], want none", err)
	}
	if len(sites) != 3 {
		t.Fatalf("Didn't find sites, want len=3, got %d", len(sites))
	}
	if sites[0] != "site1" || sites[1] != "site2" || sites[2] != "site3" {
		t.Fatalf("Didn't find sites, got [%v], [%v], [%v]", sites[0], sites[1], sites[2])
	}
}

func TestSitesForNoLocations(t *testing.T) {
	pod := `apiVersion: v1
kind: Pod
metadata:
  name: simpleapp-pod
  labels:
    app.kubernetes.io/name: SimpleApp
spec:
  containers:
  - name: myapp-container
    image: busybox:1.28
    command: ['sh', '-c', 'echo The app is running! && sleep 3600']
  initContainers:
  - name: init-myservice
    image: busybox:1.28
    command: ['sh', '-c', "sleep 2"]
`
	sites, err := SitesFor("", "", nil, []byte(pod))
	if err != nil {
		t.Fatalf("Got error [%v], want none", err)
	}
	if sites == nil || len(sites) != 0 {
		t.Fatalf("Got slice of len [%d], want non-nil empty slice", len(sites))
	}
}

func TestNoBody(t *testing.T) {
	sites, err := SitesFor("", "", nil, nil)
	if err != nil {
		t.Fatalf("Got error [%v], want none", err)
	}
	if sites == nil || len(sites) != 0 {
		t.Fatalf("Got [%v], want non-nil empty slice", sites)
	}
}

func TestConfigFor(t *testing.T) {
	kubeReply := `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "5"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{"locations":"dahu-8.grenoble.grid5000.fr,dahu-9.grenoble.grid5000.fr,dahu-23.grenoble.grid5000.fr"},"labels":{"app":"nginx"},"name":"nginx-deployment","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"image":"nginx:1.14.2","name":"nginx","ports":[{"containerPort":80}]}]}}}}
    locations: dahu-8.grenoble.grid5000.fr,dahu-9.grenoble.grid5000.fr,dahu-23.grenoble.grid5000.fr
  creationTimestamp: "2023-09-20T14:19:27Z"
  generation: 8
  labels:
    app: nginx
  name: nginx-deployment
  namespace: default
  resourceVersion: "6170"
  uid: 94c87c5d-862b-42fa-8434-e86c6c3e783c
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: nginx
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx:1.14.2
        imagePullPolicy: IfNotPresent
        name: nginx
        ports:
        - containerPort: 80
          protocol: TCP
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  conditions:
  - lastTransitionTime: "2023-09-20T14:19:27Z"
    lastUpdateTime: "2023-09-20T14:19:27Z"
    message: Deployment does not have minimum availability.
    reason: MinimumReplicasUnavailable
    status: "False"
    type: Available
  - lastTransitionTime: "2023-09-20T15:06:51Z"
    lastUpdateTime: "2023-09-20T15:06:51Z"
    message: ReplicaSet "nginx-deployment-66b6c48dd5" has timed out progressing.
    reason: ProgressDeadlineExceeded
    status: "False"
    type: Progressing
  observedGeneration: 8
  replicas: 2
  unavailableReplicas: 2
  updatedReplicas: 1`

	conf := extractCurrentConfig([]byte(kubeReply))
	expectedConf := []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{"locations":"dahu-8.grenoble.grid5000.fr,dahu-9.grenoble.grid5000.fr,dahu-23.grenoble.grid5000.fr"},"labels":{"app":"nginx"},"name":"nginx-deployment","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"image":"nginx:1.14.2","name":"nginx","ports":[{"containerPort":80}]}]}}}}`)

	if !jp.Equal(conf, expectedConf) {
		t.Fatalf("diff in current config, got %s, expected %s", conf, expectedConf)
	}
}
