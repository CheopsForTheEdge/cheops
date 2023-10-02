package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rakoo/raft"
	"github.com/rakoo/raft/raftlog"
	"github.com/rakoo/raft/transport"
	"github.com/rakoo/raft/transport/raftgrpc"

	"google.golang.org/grpc"
)

var (
	raftServer *grpc.Server
	router     *mux.Router

	raftPort   int
	raftgroups *groups
	stateDIR   string
)

func init() {
	dir, ok := os.LookupEnv("STATE_DIR")
	if !ok {
		log.Fatal("My FQDN must be given with the MYFQDN environment variable !")
	}
	stateDIR = dir
}

func Raft(port int) {
	raftPort = port
	raftgrpc.Register(
		raftgrpc.WithDialOptions(grpc.WithInsecure()),
	)
	raftgroups = &groups{
		NodeGroup: raft.NewNodeGroup(transport.GRPC),
		nodes:     make(map[uint64]*localNode),
	}

	raftServer = grpc.NewServer()
	raftgrpc.RegisterHandler(raftServer, raftgroups.Handler())

	router = mux.NewRouter()
	router.HandleFunc("/", http.HandlerFunc(save)).Methods("PUT", "POST")

	router.HandleFunc("/{groupID}", http.HandlerFunc(get)).Methods("GET")

	router.HandleFunc("/{groupID}/mgmt/nodes", http.HandlerFunc(nodes)).Methods("GET")
	// router.HandleFunc("/{groupID}/mgmt/nodes/{id}", http.HandlerFunc(removeNode)).Methods("DELETE")

	router.HandleFunc("/mgmt/groups", http.HandlerFunc(newGroup)).Methods("PUT", "POST")
	router.HandleFunc("/mgmt/groups", http.HandlerFunc(dumphttp)).Methods("GET")

	go func() {
		lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			log.Fatal(err)
		}

		err = raftServer.Serve(lis)
		if err != nil {
			log.Fatal(err)
		}
	}()

	go raftgroups.Start()
	go func() {
		err := http.ListenAndServe(":"+strconv.Itoa(port+1), router)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	raftServer.GracefulStop()
	// TODO(Shaj13) stop all nodes.
}

func dumphttp(w http.ResponseWriter, r *http.Request) {
	groups, err := dump(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(groups)
	w.Write([]byte{'\n'})
}

func dump(ctx context.Context) ([]createGroup, error) {

	raftgroups.mu.Lock()
	defer raftgroups.mu.Unlock()

	groups := make([]createGroup, 0)
	for groupID, node := range raftgroups.nodes {
		if err := node.raftnode.LinearizableRead(ctx); err != nil {
			return nil, err
		}
		peers := make([]peer, 0)
		for _, member := range node.raftnode.Members() {
			peers = append(peers, peer{
				Address: member.Address(),
				ID:      member.ID(),
			})
		}
		group := createGroup{
			GroupID: groupID,
			Peers:   peers,
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func get(w http.ResponseWriter, r *http.Request) {
	lnode, err := getNodeFromgroup(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := lnode.raftnode.LinearizableRead(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := lnode.fsm.Read()
	w.Write(value)
	w.Write([]byte{'\n'})
}

func nodes(w http.ResponseWriter, r *http.Request) {
	lnode, err := getNodeFromgroup(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	raws := []raft.RawMember{}
	membs := lnode.raftnode.Members()
	for _, m := range membs {
		raws = append(raws, m.Raw())
	}

	buf, err := json.Marshal(raws)
	if err != nil {
		panic(err)
	}

	w.Write(buf)
	w.Write([]byte{'\n'})
}

func removeNode(w http.ResponseWriter, r *http.Request) {
	lnode, err := getNodeFromgroup(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sid := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(sid, 0, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := lnode.raftnode.RemoveMember(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func newGroup(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var c createGroup
	if err := json.Unmarshal(buf, &c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	raftgroups.createAndStart(c.GroupID, c.Peers)
}

func save(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	sites, ok := r.Form["sites"]
	if !ok {
		http.Error(w, "missing sites in request", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	req, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = Save(r.Context(), sites, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

}

// Save gets the group associated to the sites and creates one if it doesn't exist
// Note: the sites will be fqdn, they won't have the raft port
func Save(ctx context.Context, sites []string, operation []byte) error {
	node := getOrCreateNodeWithSites(ctx, sites)
	if node == nil {
		return fmt.Errorf("Couldn't get node for %v", sites)
	}

	waitctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

wait:
	for {
		select {
		case <-waitctx.Done():
			return fmt.Errorf("Timeout waiting for cluster to form")
		case <-time.After(1 * time.Second):
			if node.raftnode.Leader() != 0 {
				break wait
			}
		}
	}

	rep := replicate{
		CMD:  "operation",
		Data: operation,
	}

	log.Printf("rep: %v\n", rep)

	buf, err := json.Marshal(&rep)
	if err != nil {
		return err
	}
	if err := node.raftnode.Replicate(ctx, buf); err != nil {
		log.Printf("Can't replicate group creation: %v\n", err)
		return err
	}
	return nil
}

func getOrCreateNodeWithSites(ctx context.Context, sites []string) *localNode {
	node0 := raftgroups.getNode(0)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if err := node0.raftnode.LinearizableRead(ctx); err != nil {
		return nil
	}

	findGroup := func(groups []createGroup, sites []string) (groupID, maxGroupID uint64) {
		for _, group := range groups {
			match := 0
			for _, peer := range group.Peers {
				for _, site := range sites {
					if strings.Contains(peer.Address, site) {
						match++
					}
				}
			}
			if match == len(group.Peers) {
				groupID = group.GroupID
			}
			if group.GroupID > maxGroupID {
				maxGroupID = group.GroupID
			}
		}

		return
	}

	existingGroups, err := dump(ctx)
	if err != nil {
		return nil
	}
	groupID, maxGroupID := findGroup(existingGroups, sites)

	if groupID == 0 {
		// Create group and reread

		// Find exact address and id from group 0 where everyone is
		peers := make([]peer, 0)
		node0 := raftgroups.getNode(0)

		membs := node0.raftnode.Members()
		for _, m := range membs {
			for _, site := range sites {
				if strings.Contains(m.Address(), site) {
					peers = append(peers, peer{
						Address: m.Address(),
						ID:      m.ID(),
					})
				}
			}
		}

		groupID = maxGroupID + 1

		c := createGroup{
			GroupID: groupID,
			Peers:   peers,
		}

		cbuf, err := json.Marshal(c)
		if err != nil {
			return nil
		}
		rep := replicate{
			CMD:  "groups",
			Data: cbuf,
		}

		buf, err := json.Marshal(&rep)
		if err != nil {
			return nil
		}

		if err := node0.raftnode.Replicate(ctx, buf); err != nil {
			log.Printf("Can't replicate group creation: %v\n", err)
			return nil
		}
	}

	return raftgroups.getNode(groupID)
}

func getNodeFromgroup(r *http.Request) (*localNode, error) {
	sid := mux.Vars(r)["groupID"]
	gid, err := strconv.ParseUint(sid, 0, 64)
	if err != nil {
		return nil, err
	}

	lnode := raftgroups.getNode(gid)
	if lnode == nil {
		raftgroups.mu.Lock()
		existing := make([]uint64, 0)
		for id := range raftgroups.nodes {
			existing = append(existing, id)
		}
		raftgroups.mu.Unlock()

		return nil, fmt.Errorf("group %s does not exist, we have %v", sid, existing)
	}

	return lnode, nil
}

func newstateMachine() *stateMachine {
	return &stateMachine{
		operations: make([]string, 0),
	}
}

type stateMachine struct {
	mu         sync.Mutex
	operations []string
}

func (s *stateMachine) Apply(data []byte) {
	var rep replicate
	if err := json.Unmarshal(data, &rep); err != nil {
		log.Println("unable to Unmarshal replicate", err)
		return
	}

	switch rep.CMD {
	case "operation":
		s.mu.Lock()
		defer s.mu.Unlock()
		log.Printf("Storing operation: %v\n", rep.Data)
		s.operations = append(s.operations, string(rep.Data))
	case "groups":
		var c createGroup
		err := json.Unmarshal(rep.Data, &c)
		if err != nil {
			log.Printf("Couldn't unmarshal: %v\n", err)
			break
		}
		raftgroups.createAndStart(c.GroupID, c.Peers)
	}
}

func (s *stateMachine) Snapshot() (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	buf, err := json.Marshal(&s.operations)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader(string(buf))), nil
}

func (s *stateMachine) Restore(r io.ReadCloser) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &s.operations)
	if err != nil {
		return err
	}

	return r.Close()
}

func (s *stateMachine) Read() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := json.Marshal(s.operations)
	if err != nil {
		log.Printf("Couldn't unmarshal operations: %v\n", err)
	}
	return st
}

type groups struct {
	*raft.NodeGroup
	mu    sync.Mutex
	nodes map[uint64]*localNode
}

func (g *groups) createAndStart(groupID uint64, peers []peer) {
	lg := raftlog.New(0, fmt.Sprintf("[GROUP %d]", groupID), os.Stderr, io.Discard)
	logger := raft.WithLogger(lg)

	members := make([]raft.RawMember, 1)
	for _, peer := range peers {
		if strings.Contains(peer.Address, myfqdn) {
			members[0] = raft.RawMember{
				Address: peer.Address,
				ID:      peer.ID,
			}
		} else {
			members = append(members, raft.RawMember{
				Address: peer.Address,
				ID:      peer.ID,
			})
		}
	}

	log.Printf("Creating group %d with peers %v\n", groupID, members)
	raw := raft.WithMembers(members...)
	if _, err := os.Stat(stateDIR); os.IsNotExist(err) {
		os.MkdirAll(stateDIR, 0600)
	}
	state := raft.WithStateDIR(filepath.Join(stateDIR, fmt.Sprintf("%d", groupID)))
	fallback := raft.WithFallback(
		raft.WithInitCluster(),
		raft.WithRestart(),
	)
	fsm := newstateMachine()

	node := g.Create(groupID, fsm, state, logger)

	g.mu.Lock()
	g.nodes[groupID] = &localNode{
		fsm:      fsm,
		raftnode: node,
	}
	g.mu.Unlock()

	go func() {
		err := node.Start(fallback, raw)
		if err != nil && err != raft.ErrNodeStopped {
			log.Fatal(err)
		}
	}()
}

func (g *groups) getNode(groupID uint64) *localNode {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nodes[groupID]
}

type localNode struct {
	raftnode *raft.Node
	fsm      *stateMachine
}

type replicate struct {
	CMD  string
	Data []byte
}

type createGroup struct {
	GroupID uint64
	Peers   []peer
}

type peer struct {
	Address string
	ID      uint64
}
