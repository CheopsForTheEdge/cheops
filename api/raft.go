package api

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
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

	"cheops.com/backends"
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

var (
	// Redefine this error from raftengine because we need it
	ErrNoLeader = errors.New("raft: no elected cluster leader")
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
	router.HandleFunc("/", http.HandlerFunc(get))

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
	groups, err := dump()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(groups)
	w.Write([]byte{'\n'})
}

func dump() ([]createGroup, error) {

	raftgroups.mu.Lock()
	defer raftgroups.mu.Unlock()

	groups := make([]createGroup, 0)
	for groupID, node := range raftgroups.nodes {
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
	w.WriteHeader(http.StatusCreated)
}

// Do gets the group associated to the sites and creates one if it doesn't exist
// Note: the sites will be fqdn, they won't have the raft port
// It then distributes the request in raft, waits for the reply, and gives it back
func Do(ctx context.Context, sites []string, operation Payload) (reply Payload, err error) {
	reply = Payload{}
	if len(sites) < 3 {
		return reply, fmt.Errorf("Can't save with raft, need at least three sites")
	}
	buf, err := json.Marshal(operation)
	if err != nil {
		return reply, err
	}

	node := getOrCreateNodeWithSites(ctx, sites)
	if node == nil {
		return reply, fmt.Errorf("Couldn't get node for %v", sites)
	}

	mem := make([]string, 0)
	for _, m := range node.raftnode.Members() {
		mem = append(mem, m.Address())
	}

	rep := replicate{
		CMD:  "operation",
		Data: buf,
	}

	buf2, err := json.Marshal(&rep)
	if err != nil {
		return reply, err
	}

	err = node.replicateWithRetries(ctx, buf2)
	if err != nil {
		return reply, err
	}

	replies := make([]Payload, 0, len(sites))
	for i := 0; i < len(sites); i++ {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return reply, err
			}
		case reply := <-replyBus(node.groupID, hash(buf2)):
			replies = append(replies, reply)
		case <-time.After(20 * time.Second):
			// timeout
			// happens when the node didn't do the request locally
			// but request comes from raft (could be optimized by creating
			// the channel here only, so we know)
			// or when the reply doesn't arrive locally.
			//
			// Because there are multiple cases, let's leave it like that,
			// some goroutines will wait for nothing, that's alright
			continue
		}
	}
	if len(replies) > 0 {
		// No particular reason, there is no one good response that can fit
		// while being a merge of the N replies
		// TODO: add a status header for other replies still
		return replies[0], nil
	}
	return reply, fmt.Errorf("No replies")
}

var (
	// groupId -> requestId -> chan
	replyBuses = make(map[uint64]map[string]chan Payload)
)

func replyBus(groupId uint64, requestId string) chan Payload {
	if _, ok := replyBuses[groupId]; !ok {
		replyBuses[groupId] = make(map[string]chan Payload)
	}
	if _, ok := replyBuses[groupId][requestId]; !ok {
		replyBuses[groupId][requestId] = make(chan Payload)
	}
	return replyBuses[groupId][requestId]
}

func getGroupIdForSites(ctx context.Context, sites []string) uint64 {
	log.Printf("Fetching node sites=%v\n", sites)

	findGroup := func(groups []createGroup, sites []string) (groupID uint64) {
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
		}

		return
	}

	existingGroups, err := dump()
	if err != nil {
		return 0
	}

	groupID := findGroup(existingGroups, sites)
	if groupID != 0 {
		return groupID
	} else {
		return 0
	}
}

func getOrCreateNodeWithSites(ctx context.Context, sites []string) *localNode {
	groupID := getGroupIdForSites(ctx, sites)
	if groupID != 0 {
		return raftgroups.getNode(groupID)
	}

	existingGroups, err := dump()
	if err != nil {
		return nil
	}
	max := uint64(0)
	for _, g := range existingGroups {
		if max <= g.GroupID {
			max = g.GroupID
		}
	}
	groupID = max + 1

	log.Printf("No group found, creating sites=%v\n", sites)
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

	if err := node0.replicateWithRetries(ctx, buf); err != nil {
		log.Printf("Can't replicate group creation: %v\n", err)
		return nil
	}

	return raftgroups.getNode(groupID)
}

func getNodeFromgroup(r *http.Request) (*localNode, error) {
	r.ParseForm()
	groupID := getGroupIdForSites(r.Context(), r.Form["sites"])
	if groupID != 0 {
		return raftgroups.getNode(groupID), nil
	}

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

func newstateMachine(groupID uint64) *stateMachine {
	return &stateMachine{
		groupID:    groupID,
		operations: make([]string, 0),
		replies:    make(map[string]map[string]Payload),
	}
}

type stateMachine struct {
	groupID uint64

	mu         sync.Mutex
	operations []string

	// request id -> site -> reply
	muResp  sync.Mutex
	replies map[string]map[string]Payload
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
		s.operations = append(s.operations, string(rep.Data))
		index := len(s.operations) - 1
		s.mu.Unlock()
		//s.enqueue(s.groupID, len(s.operations) - 1, rep.Data)
		go handleNewOperation(s.groupID, index, rep.Data)
	case "reply":
		var pm PayloadWithMeta
		err := json.Unmarshal(rep.Data, &pm)
		if err != nil {
			log.Printf("Couldn't unmarshal: %v\n", err)
			break
		}

		s.muResp.Lock()
		_, ok := s.replies[pm.RequestId]
		if !ok {
			s.replies[pm.RequestId] = make(map[string]Payload)
		}
		s.replies[pm.RequestId][pm.Site] = pm.Payload
		s.muResp.Unlock()

		go func() {
			replyBus(s.groupID, pm.RequestId) <- pm.Payload
		}()
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
	out := struct {
		Operations []string
		Replies    map[string]map[string]Payload
	}{
		Operations: s.operations,
		Replies:    s.replies,
	}
	st, err := json.Marshal(out)
	if err != nil {
		log.Printf("Couldn't unmarshal operations: %v\n", err)
	}
	return st
}

// enqueue puts data in the queue for this groupID with the index as a key
// for the consumer.
// Considering that data must be processed sequentially, it is expected that
// only one consumer processes it.
// The queue is a list of files, where the name is the index, and the content
// is the data. inotify manages waking up the consumer when needed
func (s *stateMachine) enqueue(groupID uint64, index int, data []byte) {
	filename := fmt.Sprintf("/var/cheops/queue/%d/%d", groupID, index)
	os.WriteFile(filename, data, 0600)
}

func handleNewOperation(groupID uint64, index int, operation []byte) {
	var p Payload
	err := json.Unmarshal(operation, &p)
	if err != nil {
		log.Printf("Couldn't unmarshal payload: %v\n", err)
		return
	}

	headerOut, bodyOut, err := backends.HandleKubernetes(p.Method, p.Path, p.Header, p.Body)
	if err != nil {
		log.Printf("Couldn't run locally: %v\n", err)
		return
	}

	resp := PayloadWithMeta{
		Payload: Payload{
			Header: headerOut,
			Body:   bodyOut,
		},
		Site:      myfqdn,
		RequestId: hash(operation),
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Couldn't marshall: %v\n", err)
		return
	}

	rep := replicate{
		CMD:  "reply",
		Data: buf,
	}

	buf2, err := json.Marshal(&rep)
	if err != nil {
		log.Printf("Couldn't marshal: %v\n", err)
	}

	node := raftgroups.getNode(groupID)
	node.replicateWithRetries(context.Background(), buf2)
}

type groups struct {
	*raft.NodeGroup
	mu    sync.Mutex
	nodes map[uint64]*localNode
}

func (g *groups) createAndStart(groupID uint64, peers []peer) {
	lg := raftlog.New(0, fmt.Sprintf("[GROUP %d]", groupID), os.Stderr, io.Discard)
	logger := raft.WithLogger(lg)

	includesMe := false
	members := make([]raft.RawMember, 1)
	for _, peer := range peers {
		if strings.Contains(peer.Address, myfqdn) {
			members[0] = raft.RawMember{
				Address: peer.Address,
				ID:      peer.ID,
			}
			includesMe = true
		} else {
			members = append(members, raft.RawMember{
				Address: peer.Address,
				ID:      peer.ID,
			})
		}
	}

	if !includesMe {
		// The group doesn't concern this site, don't actually create a node
		return
	}

	log.Printf("Creating group %d with members %v from peers %v\n", groupID, members, peers)
	raw := raft.WithMembers(members...)
	if _, err := os.Stat(stateDIR); os.IsNotExist(err) {
		os.MkdirAll(stateDIR, 0600)
	}
	state := raft.WithStateDIR(filepath.Join(stateDIR, fmt.Sprintf("%d", groupID)))
	fallback := raft.WithFallback(
		raft.WithInitCluster(),
		raft.WithRestart(),
	)
	fsm := newstateMachine(groupID)

	node := g.Create(groupID, fsm, state, logger)
	g.mu.Lock()
	g.nodes[groupID] = &localNode{
		groupID:  groupID,
		fsm:      fsm,
		raftnode: node,
	}
	g.mu.Unlock()

	go func() {
		err := node.Start(fallback, raw)
		if err != nil && err != raft.ErrNodeStopped {
			g.mu.Lock()
			delete(g.nodes, groupID)
			g.mu.Unlock()

			log.Printf("Group %d failed: %v\n", groupID, err)
		}
	}()
}

func (g *groups) getNode(groupID uint64) *localNode {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nodes[groupID]
}

type localNode struct {
	groupID  uint64
	raftnode *raft.Node
	fsm      *stateMachine
}

// replicateWithRetries tries at most 10 times to replicate the buffer
// retries are done when there is no leader because it is considered
// a transient error; another error is considered more problematic and is
// returned
func (n *localNode) replicateWithRetries(ctx context.Context, buf []byte) error {
	maxtries := 10
	for {
		err := n.raftnode.Replicate(ctx, buf)
		if err == nil {
			break
		}

		if n.raftnode.Leader() == raft.None && maxtries > 0 {
			log.Println("No leader yet, waiting 1 second")
			maxtries--
			<-time.After(1 * time.Second)
		} else {
			log.Printf("Can't replicate operation: %v\n", err)
			return err
		}
	}

	return nil
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

type PayloadWithMeta struct {
	Payload
	RequestId string
	Site      string
}

func hash(in []byte) string {
	b := sha256.Sum256(in)
	return base32.StdEncoding.EncodeToString(b[:])
}
