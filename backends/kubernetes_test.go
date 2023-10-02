package backends

import (
	"log"
	"os"
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
	kubeReply, err := os.ReadFile("kubernetes_test_all_objects.json")
	if err != nil {
		log.Fatal(err)
	}

	conf := extractCurrentConfig([]byte(kubeReply), "default:Deployment:nginx-deployment")

	expectedConf := []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{"locations":"econome-18.nantes.grid5000.fr,econome-3.nantes.grid5000.fr,econome-4.nantes.grid5000.fr"},"labels":{"app":"nginx"},"name":"nginx-deployment","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"image":"nginx:1.14.2","name":"nginx","ports":[{"containerPort":80}]}]}}}}`)

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
