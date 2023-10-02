package backends

import "testing"

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
