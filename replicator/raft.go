package replicator

import (
	"context"
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
	"cheops.com/env"
	"github.com/gorilla/mux"
	"github.com/rakoo/raft"
	"github.com/rakoo/raft/raftlog"
	"github.com/rakoo/raft/transport"
	"github.com/rakoo/raft/transport/raftgrpc"

	"google.golang.org/grpc"
)

var (
	// Redefine this error from raftengine because we need it
	ErrNoLeader = errors.New("raft: no elected cluster leader")
)

type Raft struct {
	stateDir string

	raftPort   int
	raftgroups *groups
}

func newRaft(port int) *Raft {
	dir, ok := os.LookupEnv("STATE_DIR")
	if !ok {
		log.Fatal("My FQDN must be given with the MYFQDN environment variable !")
	}

	raftgroups := &groups{
		NodeGroup: raft.NewNodeGroup(transport.GRPC),
		nodes:     make(map[uint64]*localNode),
		stateDir:  dir,
	}

	r := &Raft{
		stateDir:   dir,
		raftPort:   port,
		raftgroups: raftgroups,
	}
	r.raftgroups.handleNewOperation = r.handleNewOperation

	go r.listen()
	return r
}

func (r *Raft) listen() {
	raftgrpc.Register(
		raftgrpc.WithDialOptions(grpc.WithInsecure()),
	)

	raftServer := grpc.NewServer()
	raftgrpc.RegisterHandler(raftServer, r.raftgroups.Handler())

	router := mux.NewRouter()
	router.HandleFunc("/", http.HandlerFunc(r.dump))
	router.HandleFunc("/mgmt/groups", http.HandlerFunc(r.newGroup)).Methods("PUT", "POST")

	go func() {
		lis, err := net.Listen("tcp", ":"+strconv.Itoa(r.raftPort))
		if err != nil {
			log.Fatal(err)
		}

		err = raftServer.Serve(lis)
		if err != nil {
			log.Fatal(err)
		}
	}()

	go r.raftgroups.Start()
	go func() {
		err := http.ListenAndServe(":"+strconv.Itoa(r.raftPort+1), router)
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

type GroupDump struct {
	Id      uint64
	Members []string
	Log     []LogDump
}

type LogDump struct {
	Request Payload
	Replies []Payload
}

func (r *Raft) dump(w http.ResponseWriter, req *http.Request) {
	r.raftgroups.mu.Lock()
	defer r.raftgroups.mu.Unlock()

	groups := make([]GroupDump, 0)
	for groupID, node := range r.raftgroups.nodes {
		members := make([]string, 0)
		for _, member := range node.raftnode.Members() {
			members = append(members, member.Address())
		}
		group := GroupDump{
			Id:      groupID,
			Members: members,
		}

		node.fsm.mu.Lock()
		operations := node.fsm.operations
		node.fsm.mu.Unlock()

		node.fsm.muResp.Lock()
		allReplies := node.fsm.replies
		node.fsm.muResp.Unlock()

		logs := make([]LogDump, 0)
		for _, operation := range operations {
			replies, ok := allReplies[operation.RequestId]
			if !ok {
				replies = make([]Payload, 0)
			}

			logs = append(logs, LogDump{
				Request: operation,
				Replies: replies,
			})
		}

		group.Log = logs

		groups = append(groups, group)
	}

	json.NewEncoder(w).Encode(groups)
	w.Write([]byte{'\n'})
}

func (r *Raft) getGroups() []createGroup {
	r.raftgroups.mu.Lock()
	defer r.raftgroups.mu.Unlock()

	groups := make([]createGroup, 0)
	for groupID, node := range r.raftgroups.nodes {
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

	return groups
}

func (r *Raft) newGroup(w http.ResponseWriter, req *http.Request) {
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var c createGroup
	if err := json.Unmarshal(buf, &c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	r.raftgroups.createAndStart(c.GroupID, c.Peers)
	w.WriteHeader(http.StatusCreated)
}

// Do appends the operation to the raft log related to those sites (will create a group if
// there is none yet). Once committed each site executes the operation and writes the reply
// in the raft log. The caller will then collect the replies and return the first one that arrives
func (r *Raft) Do(ctx context.Context, sites []string, operation Payload) (reply Payload, err error) {
	reply = Payload{}
	if len(sites) < 3 {
		return reply, fmt.Errorf("Can't save with raft, need at least three sites")
	}

	buf, err := json.Marshal(operation)
	if err != nil {
		return reply, err
	}

	node := r.getOrCreateNodeWithSites(ctx, sites)
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

wait:
	for i := 0; i < len(sites); i++ {
		log.Printf("Waiting for reply %d\n", i)
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return reply, err
			}
		case reply := <-bus.For(node.groupID, operation.RequestId):
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
			break wait
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

func (r *Raft) getGroupIdForSites(ctx context.Context, sites []string) uint64 {
	log.Printf("Fetching node sites=%v\n", sites)

	existingGroups := r.getGroups()
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

	groupID := findGroup(existingGroups, sites)
	if groupID != 0 {
		return groupID
	} else {
		return 0
	}
}

func (r *Raft) getOrCreateNodeWithSites(ctx context.Context, sites []string) *localNode {
	groupID := r.getGroupIdForSites(ctx, sites)
	if groupID != 0 {
		return r.raftgroups.getNode(groupID)
	}

	existingGroups := r.getGroups()
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
	node0 := r.raftgroups.getNode(0)

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

	return r.raftgroups.getNode(groupID)
}

func (r *Raft) getNodeFromgroup(req *http.Request) (*localNode, error) {
	req.ParseForm()
	groupID := r.getGroupIdForSites(req.Context(), req.Form["sites"])
	if groupID != 0 {
		return r.raftgroups.getNode(groupID), nil
	}

	sid := mux.Vars(req)["groupID"]
	gid, err := strconv.ParseUint(sid, 0, 64)
	if err != nil {
		return nil, err
	}

	lnode := r.raftgroups.getNode(gid)
	if lnode == nil {
		r.raftgroups.mu.Lock()
		existing := make([]uint64, 0)
		for id := range r.raftgroups.nodes {
			existing = append(existing, id)
		}
		r.raftgroups.mu.Unlock()

		return nil, fmt.Errorf("group %s does not exist, we have %v", sid, existing)
	}

	return lnode, nil
}

func newstateMachine(groupID uint64, handleNewOperation func(groupID uint64, payload Payload), handleNewGroup func(groupID uint64, peers []peer)) *stateMachine {
	return &stateMachine{
		groupID:            groupID,
		operations:         make([]Payload, 0),
		replies:            make(map[string][]Payload),
		handleNewOperation: handleNewOperation,
	}
}

type stateMachine struct {
	groupID uint64

	mu         sync.Mutex
	operations []Payload

	// request id -> site -> reply
	muResp  sync.Mutex
	replies map[string][]Payload

	// handleNewOperation is called whenever a new operation is registered
	handleNewOperation func(groupID uint64, payload Payload)

	// handleNewGroup is called whenever a new group creation request is received
	handleNewGroup func(groupID uint64, peers []peer)
}

func (s *stateMachine) Apply(data []byte) {
	var rep replicate
	if err := json.Unmarshal(data, &rep); err != nil {
		log.Println("unable to Unmarshal replicate", err)
		return
	}

	switch rep.CMD {
	case "operation":
		var p Payload
		err := json.Unmarshal(rep.Data, &p)
		if err != nil {
			log.Printf("Couldn't unmarshal: %v\n", err)
			break
		}

		log.Printf("Received request %v\n", p.RequestId)

		s.mu.Lock()
		s.operations = append(s.operations, p)
		s.mu.Unlock()
		//s.enqueue(s.groupID, len(s.operations) - 1, rep.Data)
		go s.handleNewOperation(s.groupID, p)
	case "reply":
		var p Payload
		err := json.Unmarshal(rep.Data, &p)
		if err != nil {
			log.Printf("Couldn't unmarshal: %v\n", err)
			break
		}

		log.Printf("Received reply to %v from %v\n", p.RequestId, p.Site)

		s.muResp.Lock()
		_, ok := s.replies[p.RequestId]
		if !ok {
			s.replies[p.RequestId] = make([]Payload, 0)
		}
		s.replies[p.RequestId] = append(s.replies[p.RequestId], p)
		s.muResp.Unlock()

		go func() {
			bus.For(s.groupID, p.RequestId) <- p
		}()
	case "groups":
		var c createGroup
		err := json.Unmarshal(rep.Data, &c)
		if err != nil {
			log.Printf("Couldn't unmarshal: %v\n", err)
			break
		}
		s.handleNewGroup(c.GroupID, c.Peers)
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

func (r *Raft) handleNewOperation(groupID uint64, p Payload) {
	headerOut, bodyOut, err := backends.HandleKubernetes(p.Method, p.Path, p.Header, p.Body)
	if err != nil {
		log.Printf("Couldn't run locally: %v\n", err)
		return
	}

	resp := Payload{
		RequestId: p.RequestId,
		Header:    headerOut,
		Body:      bodyOut,
		Site:      env.Myfqdn,
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

	node := r.raftgroups.getNode(groupID)
	node.replicateWithRetries(context.Background(), buf2)
}

type groups struct {
	*raft.NodeGroup
	mu    sync.Mutex
	nodes map[uint64]*localNode

	handleNewOperation func(groupID uint64, payload Payload)
	stateDir           string
}

func (g *groups) createAndStart(groupID uint64, peers []peer) {
	lg := raftlog.New(0, fmt.Sprintf("[GROUP %d]", groupID), os.Stderr, io.Discard)
	logger := raft.WithLogger(lg)

	includesMe := false
	members := make([]raft.RawMember, 1)
	for _, peer := range peers {
		if strings.Contains(peer.Address, env.Myfqdn) {
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
	if _, err := os.Stat(g.stateDir); os.IsNotExist(err) {
		os.MkdirAll(g.stateDir, 0600)
	}
	state := raft.WithStateDIR(filepath.Join(g.stateDir, fmt.Sprintf("%d", groupID)))
	fallback := raft.WithFallback(
		raft.WithInitCluster(),
		raft.WithRestart(),
	)
	fsm := newstateMachine(groupID, g.handleNewOperation, g.createAndStart)

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
			if err := n.raftnode.LinearizableRead(ctx); err != nil {
				log.Printf("Can't make node linearizable read: %v\n", err)
			}
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

type replyBus struct {
	// groupId -> requestId -> chan
	mu    sync.Mutex
	buses map[uint64]map[string]chan Payload
}

var (
	bus = &replyBus{
		buses: make(map[uint64]map[string]chan Payload),
	}
)

func (rb *replyBus) For(groupId uint64, requestId string) chan Payload {
	rb.mu.Lock()
	if _, ok := rb.buses[groupId]; !ok {
		rb.buses[groupId] = make(map[string]chan Payload)
	}
	if _, ok := rb.buses[groupId][requestId]; !ok {
		rb.buses[groupId][requestId] = make(chan Payload)
	}
	c := rb.buses[groupId][requestId]
	rb.mu.Unlock()

	return c
}
