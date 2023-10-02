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
	"github.com/shaj13/raft"
	"github.com/shaj13/raft/raftlog"
	"github.com/shaj13/raft/transport"
	"github.com/shaj13/raft/transport/raftgrpc"
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

	// router.HandleFunc("/{groupID}/{key}", http.HandlerFunc(get)).Methods("GET")

	router.HandleFunc("/{groupID}/mgmt/nodes", http.HandlerFunc(nodes)).Methods("GET")
	// router.HandleFunc("/{groupID}/mgmt/nodes/{id}", http.HandlerFunc(removeNode)).Methods("DELETE")

	router.HandleFunc("/mgmt/groups", http.HandlerFunc(newGroup)).Methods("PUT", "POST")

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

	value := lnode.fsm.ReadOperations()
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

	rep := replicate{
		CMD:  "operation",
		Data: operation,
	}

	buf, err := json.Marshal(&rep)
	if err != nil {
		return err
	}
	if err := node.raftnode.Replicate(ctx, buf); err != nil {
		log.Printf("Can't replicate group creation: %v\n", err)
		return nil
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

	groupID, maxGroupID := findGroup(node0.fsm.ReadGroups(), sites)

	if groupID == 0 {
		// Create group and reread

		// Find exact address and id from group 0 where everyone is
		peers := make([]peer, 0)
		node0 := raftgroups.getNode(0)

		membs := node0.raftnode.Members()
		for _, m := range membs {
			peers = append(peers, peer{
				Address: m.Address(),
				ID:      m.ID(),
			})
		}
		newgroup := createGroup{
			GroupID: maxGroupID + 1,
			Peers:   peers,
		}
		buf, err := json.Marshal(&newgroup)
		if err != nil {
			log.Printf("Can't marshal content: %v\n", newgroup)
		}

		rep := replicate{
			CMD:  "groups",
			Data: buf,
		}

		buf, err = json.Marshal(&rep)
		if err != nil {
			log.Printf("Can't marshal content: %v\n", rep)
			return nil
		}

		if err := node0.raftnode.Replicate(ctx, buf); err != nil {
			log.Printf("Can't replicate group creation: %v\n", err)
			return nil
		}

		groupID, _ = findGroup(node0.fsm.ReadGroups(), sites)
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
		return nil, fmt.Errorf("group %s does not exist", sid)
	}

	return lnode, nil
}

func newstateMachine() *stateMachine {
	return &stateMachine{
		operations: make([]string, 0),
	}
}

type stateMachine struct {

	// group >0 store operations
	// group 0 doesn't store operations
	omu        sync.Mutex
	operations []string

	// group 0 stores all existing groups, including itself
	// only group 0 does that
	gmu    sync.Mutex
	groups []createGroup
}

func (s *stateMachine) Apply(data []byte) {
	var rep replicate
	if err := json.Unmarshal(data, &rep); err != nil {
		log.Println("unable to Unmarshal replicate", err)
		return
	}

	switch rep.CMD {
	case "operation":
		s.omu.Lock()
		defer s.omu.Unlock()
		s.operations = append(s.operations, string(rep.Data))
	case "groups":
		s.gmu.Lock()
		defer s.gmu.Unlock()

		var c createGroup
		err := json.Unmarshal(rep.Data, &c)
		if err != nil {
			log.Println("Can't unmarshal: %v\n", err)
		}

		s.groups = append(s.groups, c)

	}
}

func (s *stateMachine) Snapshot() (io.ReadCloser, error) {
	s.omu.Lock()
	defer s.omu.Unlock()
	buf, err := json.Marshal(&s.operations)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader(string(buf))), nil
}

func (s *stateMachine) Restore(r io.ReadCloser) error {
	s.omu.Lock()
	defer s.omu.Unlock()

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

func (s *stateMachine) ReadOperations() []byte {
	s.omu.Lock()
	defer s.omu.Unlock()
	st, _ := json.Marshal(s.operations)
	return st
}

func (s *stateMachine) ReadGroups() []createGroup {
	s.gmu.Lock()
	defer s.gmu.Unlock()

	r := make([]createGroup, 0, len(s.groups))
	for _, group := range s.groups {
		r = append(r, group)
	}

	return r
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
	fsm.groups = append(fsm.groups, createGroup{
		GroupID: groupID,
		Peers:   peers,
	})

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
