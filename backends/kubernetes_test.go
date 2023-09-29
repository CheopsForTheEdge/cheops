package backends

import (
	"testing"

	jp "github.com/evanphx/json-patch"
)

func TestConfigFor(t *testing.T) {
	kubeReply := `apiVersion: v1
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    annotations:
      deployment.kubernetes.io/revision: "4"
      kubectl.kubernetes.io/last-applied-configuration: |
        {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{"locations":"dahu-8.grenoble.grid5000.fr,dahu-9.grenoble.grid5000.fr,dahu-23.grenoble.grid5000.fr"},"labels":{"app":"nginx"},"name":"nginx-deployment","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"image":"nginx:1.14.2","name":"nginx","ports":[{"containerPort":90}]}]}}}}
      locations: dahu-8.grenoble.grid5000.fr,dahu-9.grenoble.grid5000.fr,dahu-23.grenoble.grid5000.fr
    creationTimestamp: "2023-09-20T15:47:01Z"
    generation: 6
    labels:
      app: nginx
    name: nginx-deployment
    namespace: default
    resourceVersion: "9873"
    uid: 1e5a2c12-718a-4497-84b0-923b4285a9e8
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
          - containerPort: 90
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
    - lastTransitionTime: "2023-09-20T15:47:01Z"
      lastUpdateTime: "2023-09-20T15:47:01Z"
      message: Deployment does not have minimum availability.
      reason: MinimumReplicasUnavailable
      status: "False"
      type: Available
    - lastTransitionTime: "2023-09-20T15:47:01Z"
      lastUpdateTime: "2023-09-20T15:54:50Z"
      message: ReplicaSet "nginx-deployment-ffc7dcfbb" is progressing.
      reason: ReplicaSetUpdated
      status: "True"
      type: Progressing
    observedGeneration: 6
    replicas: 2
    unavailableReplicas: 2
    updatedReplicas: 1
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
`

	conf := extractCurrentConfig([]byte(kubeReply))
	expectedConf := []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{"locations":"dahu-8.grenoble.grid5000.fr,dahu-9.grenoble.grid5000.fr,dahu-23.grenoble.grid5000.fr"},"labels":{"app":"nginx"},"name":"nginx-deployment","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"image":"nginx:1.14.2","name":"nginx","ports":[{"containerPort":90}]}]}}}}`)

	if !jp.Equal(conf, expectedConf) {

		diff, _ := jp.CreateMergePatch(expectedConf, conf)
		t.Fatalf("diff in current config, diff is %s, expected %s", diff, expectedConf)
	}
}

func TestResourceIdForWithDefaultNamespace(t *testing.T) {
	input := `apiVersion: v1
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

	id, err := ResourceIdFor("", "", nil, []byte(input))
	if err != nil {
		t.Fatal(err)
	}

	expected := "default:Pod:simpleapp-pod"
	if id != expected {
		t.Fatalf("Wrong id: got [%s], expected [%s]", id, expected)
	}
}

func TestResourceIdForWithSpecificNamespace(t *testing.T) {
	input := `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    k8s-app: flannel
  name: flannel
  namespace: kube-flannel`

	id, err := ResourceIdFor("", "", nil, []byte(input))
	if err != nil {
		t.Fatal(err)
	}

	expected := "kube-flannel:ServiceAccount:flannel"
	if id != expected {
		t.Fatalf("Wrong id: got [%s], expected [%s]", id, expected)
	}
}
