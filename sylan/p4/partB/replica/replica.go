package main

import (
	"encoding/json"
	"fmt"
	"github.com/looplab/tarjan"
	. "github.com/mattn/go-getopt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	//"runtime"
	"strconv"
	"strings"
	"syscall"

	//"sync"
	"time"

	"818fall18/sylan/p4/partB/pb" /* MODIFY ME */

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/reflection"
)

const LOG_MAX = 1000000
const HT_INIT_SIZE = 20000

type server struct{}

//=====================================================================

type Metrics struct {
	Arguments       string
	N               int64
	Sends, Receives int
	MessageSends    map[string]int64
	MessageReceives map[string]int64
	Slows, Fasts    int
	//	Ops             int64
	Aggregated int64
	GraphEdges int64
	ExecCands  int64
	Blips      int64 // dependence on something we have not yet seen
}

//=====================================================================
type ConfigAddr struct {
	Host string
	Port string
}

type ConfigFile struct {
	Replicas []ConfigAddr
}

//=====================================================================

type MsgContainer struct {
	clientOp   *pb.Operation
	clientChan chan *pb.ProposeReply
	PeerMsg    *pb.PeerRequest
}

var (
	N           int
	gSeq        = 0
	me          = 0
	debug       = 0
	aggregating = false
	configFile  = "../config.json"
	config      = new(ConfigFile)

	clients []pb.ReplicaClient

	replyCount    = make(map[int64]int)
	serverMode    = false
	committed     = make(map[int64]bool)
	allReps       []int
	asyncExec     = true
	bigchan       = make(chan *MsgContainer, 1000)
	commitPause   = false
	conflicts     []map[string]int64
	executed      []int64
	fastQ         int
	hostname      string
	instancesExec []int
	instancesSent []int
	logs          [][]*pb.Instance
	triggers      = make(map[int]chan bool)
	maxSeq        = 0
	metrics       Metrics
	outChannels   []chan *pb.PeerRequest // out channel to other replicas
	outfile       = false
	slowQ         int
	store         = make(map[string][]byte)
	streams       []pb.Replica_DispatchClient // out stream fed by 'outchannels'
	thrifty       = true
	thriftyReps   []int
	kvmap         = make(map[string][]byte)
)

//=====================================================================

func main() {
	hostname, _ = os.Hostname()

	metrics.MessageReceives = make(map[string]int64)
	metrics.MessageSends = make(map[string]int64)
	for {
		if c := Getopt("ad:r:N:oc:tep"); c == EOF {
			break
		} else {
			switch c {
			case 'a':
				aggregating = !aggregating

			case 'd':
				debug, _ = strconv.Atoi(OptArg)

			case 'N':
				N, _ = strconv.Atoi(OptArg)

			case 'o':
				outfile = !outfile

			case 'r':
				me, _ = strconv.Atoi(OptArg)
				p_out(string(me))

			case 'e':
				asyncExec = !asyncExec

			case 'p':
				commitPause = !commitPause

			case 't':
				thrifty = !thrifty

			case 'c':
				configFile = OptArg
			}
		}
	}

	if jsonFile, err := os.Open(configFile); err != nil {
		fmt.Println(err)
	} else {
		defer jsonFile.Close()
		byteValue, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			fmt.Println(err)
		}

		err = json.Unmarshal([]byte(byteValue), &config)
		if err != nil {
			fmt.Println(err)
		}
	}

	//=====================================================================
	// All messages to a single other replica go though a single outgoing
	// 1) Dial each remote replica, retrying on error (other replica might be slow).
	// 2) Set up 'outChannels[i]', and a go thread to read from it and write to stream.

	// thrifty
	serverMode = true
	executed = make([]int64, N)
	for i := 0; i < N; i++ {
		executed[i] = -1
	}
	if thrifty {
		thriftyReps = make([]int, N/2)
		for i := 1; i <= N/2; i++ {
			thriftyReps[i-1] = (i + me) % N
		}
	} else {
		thriftyReps = make([]int, N-1)
		for i := 0; i < N; i++ {
			count := 0
			if i != me {
				thriftyReps[count] = i
				count = count + 1
			}
		}
	}

	allReps = make([]int, N)
	logs = make([][]*pb.Instance, N)
	for i := 0; i < N; i++ {
		allReps[i] = i
		logs[i] = make([]*pb.Instance, 0)
	}

	//fmt.Println(config.Replicas[0].Host)
	outChannels = make([]chan *pb.PeerRequest, N)

	clients = make([]pb.ReplicaClient, N)
	for i := 0; i < N; i++ {
		addr := config.Replicas[i].Host + ":" + config.Replicas[i].Port
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			panic(fmt.Sprintf("did no t connect: %v", err))
		}
		//p_err("Dialed %q\n", addr)
		clients[i] = pb.NewReplicaClient(conn)
		outChannels[i] = make(chan *pb.PeerRequest)
		go func(k uint64) {
			var (
				stream pb.Replica_DispatchClient
				err    error
			)

			firstTime := true
			for req := range outChannels[k] {
				if firstTime {
					firstTime = false
					stream, err = clients[k].Dispatch(context.TODO())
					if err != nil {
						fmt.Println(err)
					}
				}
				p_level(1, "Sending msg \"%q\" to %v\n", pb.MsgType_name[int32(req.Type)], k)

				switch q := (req.Message).(type) {
				case *pb.PeerRequest_Preaccept:
					p_level(2, "preaccept deps: %v, seq: %v\n", q.Preaccept.Inst.Deps, q.Preaccept.Inst.Seq)
				case *pb.PeerRequest_Preacceptreply:
					p_level(2, "preacceptReply deps: %v, seq: %v\n", logs[k][q.Preacceptreply.Slot].Deps, q.Preacceptreply.Seq)
				case *pb.PeerRequest_Accept:
					p_level(2, "accept deps: %v, seq: %v\n", q.Accept.Inst.Deps, q.Accept.Inst.Seq)
				case *pb.PeerRequest_Acceptreply:
					p_level(2, "replyAccept deps: %v, seq: %v\n", logs[k][q.Acceptreply.Slot].Deps, logs[k][q.Acceptreply.Slot].Seq)
				}
				//fmt.Println(req)
				stream.Send(req)
			}
		}(uint64(i))
	}

	/*
		go func() {
			for {
				for _, c := range outChannels {
					p_out("Len of outChannels: %d\n", len(c))
				}
				time.Sleep(time.Second * 3)
			}
		}()
	*/

	// start our big dispatcher
	go bigDispatcher()

	//=====================================================================
	// possibly redirect stdout to a file
	oldStdout := os.Stdout
	temp, _ := os.Create(hostname + "." + strconv.Itoa(me))
	if outfile {
		p_err("STDOUT now set to %q\n", hostname+"."+strconv.Itoa(me))
		os.Stdout = temp
		os.Stderr = temp
		p_err("blah\n")
	}

	//=====================================================================
	// catch termination, write metrics to a file
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGTERM, os.Interrupt, os.Kill)
	go func() {
		<-sigch

		b, _ := json.MarshalIndent(metrics, "   ", "   ")
		//p_err("\n\n%s\n%v, exec%v\n", string(b), logs[0][:3], executed)

		os.Stdout = oldStdout
		temp.Sync()
		temp.Close()
		ioutil.WriteFile(fmt.Sprintf("output.%d", me), b, 0644)
		p_exit("out\n")
	}()

	metrics.Arguments = strings.Join(os.Args, " ")

	//=====================================================================
	// start execute thread

	//=====================================================================
	// listen to incoming messages socket for msgs from other replicas
	// create gRPC server, register it. See the tutorial.
	grpcServer := grpc.NewServer()
	pb.RegisterReplicaServer(grpcServer, &server{})
	l, err := net.Listen("tcp", ":"+string(config.Replicas[me].Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	p_err("Ready... rep %d, port %q, N=%v, exec=%v, pause=%v, agg=%v, reps:%v\n", me, config.Replicas[me].Port, N, asyncExec, commitPause, aggregating, thriftyReps)
	grpcServer.Serve(l)

	p_exit("Out\n")
}

//=====================================================================

/*
func (s *server) Exit(ctx context.Context, req *pb.ExitRequest) (*pb.ExitReply, error) {
	p_out("EXIT msg\n")
	go func() {
		b, _ := json.MarshalIndent(metrics, "   ", "   ")
		ioutil.WriteFile(fmt.Sprintf("output.%d", me), b, 0644)
		p_err("\n\n%s\nMetrics.Executed:%v\n", string(b), executed)
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}()
	return &pb.ExitReply{}, nil
}
*/

// example of convenience function
func wrapPreacceptRequest(req *pb.PreacceptRequest) *pb.PeerRequest {
	return &pb.PeerRequest{
		Type: pb.MsgType_PREACCEPT,
		From: int64(me),
		Message: &pb.PeerRequest_Preaccept{
			Preaccept: req,
		},
	}
}

func wrapPreacceptReply(req *pb.PreacceptReply) *pb.PeerRequest {
	return &pb.PeerRequest{
		Type: pb.MsgType_PREACCEPTREPLY,
		From: int64(me),
		Message: &pb.PeerRequest_Preacceptreply{
			Preacceptreply: req,
		},
	}
}

func max(x int, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

func wrapAcceptRequest(req *pb.AcceptRequest) *pb.PeerRequest {
	return &pb.PeerRequest{
		Type: pb.MsgType_ACCEPT,
		From: int64(me),
		Message: &pb.PeerRequest_Accept{
			Accept: req,
		},
	}
}

func wrapAcceptReply(req *pb.AcceptReply) *pb.PeerRequest {
	return &pb.PeerRequest{
		Type: pb.MsgType_ACCEPTREPLY,
		From: int64(me),
		Message: &pb.PeerRequest_Acceptreply{
			Acceptreply: req,
		},
	}
}

func wrapCommitRequest(req *pb.CommitRequest) *pb.PeerRequest {
	return &pb.PeerRequest{
		Type: pb.MsgType_COMMIT,
		From: int64(me),
		Message: &pb.PeerRequest_Commit{
			Commit: req,
		},
	}
}

//=====================================================================

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.ClientReply, error) {
	p_out("received GET %q (%q), about to send request to bigchan\n",
		req.Key, req.Identity)

	op := new(pb.Operation)
	op.Type = pb.AccessType_ACCESS_READ
	op.Key = req.Key
	op.Identity = req.Identity

	rep, _ := Propose(op)

	return &pb.ClientReply{Pair: &pb.KVPair{Key: req.Key, Value: rep.Value}}, nil
}

func (s *server) Put(ctx context.Context, req *pb.PutRequest) (*pb.ClientReply, error) {
	p_out("received PUT %q/%q (%q), about to send request to bigchan\n",
		req.Key, req.Value, req.Identity)

	op := new(pb.Operation)
	op.Type = pb.AccessType_ACCESS_WRITE
	op.Key = req.Key
	op.Value = req.Value
	op.Identity = req.Identity

	Propose(op)

	return &pb.ClientReply{}, nil
}

func (s *server) Del(ctx context.Context, req *pb.DelRequest) (*pb.ClientReply, error) {
	p_out("received DEL %q (%q), about to send request to bigchan\n",
		req.Key, req.Identity)

	op := new(pb.Operation)
	op.Type = pb.AccessType_ACCESS_DEL
	op.Key = req.Key

	Propose(op)

	return &pb.ClientReply{}, nil
}

// Propose is a unary RPC that allows clients to propose Instances to the quorum.
func Propose(op *pb.Operation) (*pb.ProposeReply, error) {

	msg := new(MsgContainer)

	msg.clientOp = op
	if op.Type == pb.AccessType_ACCESS_READ || op.Type == pb.AccessType_ACCESS_WRITEREAD {
		replyChan := make(chan *pb.ProposeReply)
		msg.clientChan = replyChan
		bigchan <- msg
		reply := <-replyChan
		return reply, nil
	} else {
		bigchan <- msg
		return nil, nil
	}

}

//=====================================================================

// return sorted (by seq#'s) list of SCCs, each sorted (also by seq#'s) correctly

type node struct {
	rep  int
	slot int
}

type nodes []node

func (s nodes) Len() int {
	return len(s)
}

func (s nodes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func nodeLess(x, y node) bool {
	if logs[x.rep][x.slot].Seq != logs[y.rep][y.slot].Seq {
		return logs[x.rep][x.slot].Seq < logs[y.rep][y.slot].Seq
	} else {
		return y.rep < y.rep
	}
}

func (s nodes) Less(i, j int) bool {
	return nodeLess(s[i], s[j])
}

func checkValid(n node) bool {
	if n.slot >= 0 && len(logs[n.rep]) > n.slot && logs[n.rep][n.slot] != nil {
		return true
	} else {
		return false
	}
}

func executeSome() [][]*pb.Instance {

	graph := make(map[interface{}][]interface{})
	/*
		fmt.Print(string(me) + "  ")
		fmt.Println(logs)
	*/

	// find unexecuted Instances by using last cmd from each rep as roots
	for repIdx, repLogs := range logs {
		for slot, inst := range repLogs {
			curNode := node{rep: repIdx, slot: slot}
			if checkValid(curNode) {
				deps := make([]interface{}, 0)
				for depRepIdx, depSlot := range inst.Deps {
					depNode := node{rep: int(depRepIdx), slot: int(depSlot)}
					if checkValid(depNode) {
						if depSlot >= 0 {
							deps = append(deps, node{rep: int(depRepIdx), slot: int(depSlot)})
						}
					}
				}
				graph[curNode] = deps
			}
		}
	}

	// make graph

	// call tarjan
	retVal := make([][]*pb.Instance, N)

	// sort the SSC's
	ordered := tarjan.Connections(graph)

	// sort by seq and repid
	for layerIdx, scc := range ordered {
		o := make(nodes, len(ordered[layerIdx]))
		for idx, n := range scc {
			o[idx] = n.(node)
		}
		sort.Sort(o)
		for idx, n := range o {
			ordered[layerIdx][idx] = n
		}
	}

	breakFlag := false
	for _, scc := range ordered {
		for _, _n := range scc {
			n := _n.(node)
			//p_out("rep:%d slot:%d\n", n.rep, n.slot)
			if logs[n.rep][n.slot].State == pb.InstState_COMMITTED {
				for rep, slot := range logs[n.rep][n.slot].Deps {
					if slot >= 0 && (logs[rep][slot] == nil || (logs[rep][slot].State != pb.InstState_COMMITTED && logs[rep][slot].State != pb.InstState_EXECUTED)) {
						p_out("deps: %d %d\n", rep, slot)
						breakFlag = true
						break
					}
				}
				if breakFlag {
					break
				}
				//p_out("Execute rep: %d, slot: %d\n", inst.Rep, inst.Slot)
				retVal[n.rep] = append(retVal[n.rep], logs[n.rep][n.slot])
				execute(logs[n.rep][n.slot])
				if executed[n.rep] == -1 || nodeLess(node{rep: n.rep, slot: int(executed[n.rep])}, node{rep: n.rep, slot: n.slot}) {
					executed[n.rep] = int64(n.slot)
				}
			}
		}
		if breakFlag {
			break
		}
	}

	return retVal
}

func batching(queue []*MsgContainer) {
	//p_out("batching")
	ops := make([]*pb.Operation, len(queue))
	for idx, msg := range queue {
		ops[idx] = msg.clientOp
	}
	slot := handleProposition(ops)
	if _, ok := triggers[int(slot)]; !ok {
		triggers[int(slot)] = make(chan bool)
	}

	go func(queue []*MsgContainer, slot int64) {
		//p_out("wait for trigger rep: %d, slot: %d\n", me, slot)
		<-triggers[int(slot)]
		//p_out("receive trigger\n")
		for idx, e := range queue {
			if e.clientChan != nil {
				proposeReply := new(pb.ProposeReply)
				proposeReply.Slot = slot
				proposeReply.Value = logs[me][slot].Ops[idx].Value
				e.clientChan <- proposeReply
			}
		}
	}(queue, slot)

}

func execute(inst *pb.Instance) {
	for _, op := range inst.Ops {
		switch op.Type {
		case pb.AccessType_ACCESS_READ:
			if val, ok := kvmap[op.Key]; ok {
				op.Value = val
			} else {
				op.Value = nil
			}
		case pb.AccessType_ACCESS_WRITE:
			kvmap[op.Key] = op.Value
		case pb.AccessType_ACCESS_DEL:
			delete(kvmap, op.Key)
		}
	}
	inst.State = pb.InstState_EXECUTED
	if serverMode && me == int(inst.Rep) {
		//p_out("me: %d, rep: %d sends trigger slot: %d\n", me, int(inst.Rep), int(inst.Slot))
		triggers[int(inst.Slot)] <- true
	}
}

func bigDispatcher() {

	eTicker := time.NewTicker(5 * time.Millisecond)
	//proposalChannels := make(map[string]chan *pb.ProposeReply, LOG_MAX) // to proposer

	queue := make([]*MsgContainer, 0)

	for {
		select {
		case e := <-bigchan:
			if e.clientOp != nil {
				p_out("pmsg %q from %v\n", pb.MsgType_name[int32(pb.MsgType_PROPOSE)], e.clientOp.Identity)
				// clientRequest
				queue = append(queue, e)

			} else {

				if len(queue) > 0 {
					batching(queue)
					queue = make([]*MsgContainer, 0)
				}

				p_out("pmsg %q from %v\n", pb.MsgType_name[int32(e.PeerMsg.Type)], e.PeerMsg.From)
				// peerRequest
				pReq := e.PeerMsg
				switch req := (pReq.Message).(type) {
				case *pb.PeerRequest_Preaccept:
					handlePreacceptRequest(pReq.From, req.Preaccept)
				case *pb.PeerRequest_Preacceptreply:
					handlePreacceptReply(pReq.From, req.Preacceptreply)
				case *pb.PeerRequest_Accept:
					handleAcceptRequest(pReq.From, req.Accept)
				case *pb.PeerRequest_Acceptreply:
					handleAcceptReply(pReq.From, req.Acceptreply)
				case *pb.PeerRequest_Commit:
					handleCommit(pReq.From, req.Commit)
				}
			}
		case <-eTicker.C:
			executeSome()

		default:
			if len(queue) > 0 {
				batching(queue)
				queue = make([]*MsgContainer, 0)
			}
			time.Sleep(time.Second)
		}
	}

	p_err("You shoud never be here.")

}

func (s *server) Dispatch(instream pb.Replica_DispatchServer) (err error) {
	for {
		req, err := instream.Recv()
		//p_out("Dispatch receives a value\n")
		if err == io.EOF {
			return nil
		} else if err != nil {
			fmt.Println(req)
			fmt.Println(err)
			return err
		}
		msg := new(MsgContainer)
		msg.clientOp = nil
		msg.PeerMsg = req
		// encapsulate message and stuff down bigChannel
		bigchan <- msg
	}
}

func findMaxConflict(ops []*pb.Operation) (int, []int64) {
	maxSeq := 0
	deps := make([]int64, N)
	for _, i := range allReps {
		deps[i] = -1
	}
	for rep, repLogs := range logs {
		for i := len(repLogs) - 1; i >= 0; i-- {
			conflict := false
			for _, op0 := range repLogs[i].Ops {
				for _, op1 := range ops {
					if op0.Key == op1.Key {
						conflict = true
						break
					}
				}
			}
			if conflict && int(repLogs[i].Seq) > maxSeq {
				maxSeq = int(repLogs[i].Seq)
				deps[rep] = int64(i)
			}
		}
	}
	return maxSeq, deps
}

// in bigchan, so no mutex
func handleProposition(ops []*pb.Operation) int64 {

	maxSeq, deps := findMaxConflict(ops)
	inst := new(pb.Instance)
	inst.Rep = int64(me)
	inst.Seq = int64(maxSeq + 1)
	inst.Slot = int64(len(logs[me]))
	inst.Deps = deps
	inst.Ops = ops
	preacceptReq := new(pb.PreacceptRequest)
	inst.State = pb.InstState_PREACCEPTED
	preacceptReq.Inst = inst

	logs[inst.Rep] = append(logs[inst.Rep], inst)

	for _, i := range thriftyReps {
		peerRequest := wrapPreacceptRequest(preacceptReq)
		//p_out("send to channels %d\n", i)
		outChannels[i%N] <- peerRequest
	}

	// TODO
	//updateConflicts(inst)

	return inst.Slot
}

func handlePreacceptRequest(from int64, req *pb.PreacceptRequest) *pb.PeerRequest {
	inst := req.Inst

	maxSeq, deps := findMaxConflict(inst.Ops)
	if inst.Seq != int64(maxSeq+1) {
		inst.Seq = int64(max(int(inst.Seq), maxSeq+1))
		inst.Changed = true
	}
	for i := 0; i < N; i++ {
		if inst.Deps[i] != deps[i] {
			inst.Deps[i] = int64(max(int(inst.Deps[i]), int(deps[i])))
			inst.Changed = true
		}
	}

	if len(logs[inst.Rep]) <= int(inst.Slot) {
		padSlice := make([]*pb.Instance, int(inst.Slot)+1-len(logs[inst.Rep]))
		logs[inst.Rep] = append(logs[inst.Rep], padSlice...)
	}
	logInst := *inst
	logs[inst.Rep][inst.Slot] = &logInst

	reply := new(pb.PreacceptReply)
	reply.Slot = inst.Slot
	reply.Seq = inst.Seq
	reply.Deps = inst.Deps
	reply.Changed = inst.Changed
	peerRequest := wrapPreacceptReply(reply)
	outChannels[from] <- peerRequest
	return peerRequest
}

func commit(rep int, slot int64) {
	if commitPause {
		time.Sleep(time.Second * 10)
	}
	logInst := logs[rep][slot]
	logInst.State = pb.InstState_COMMITTED
	commitReq := new(pb.CommitRequest)
	inst := *logInst
	commitReq.Inst = &inst
	pReq := wrapCommitRequest(commitReq)
	for _, rep := range allReps {
		if rep != me {
			outChannels[rep] <- pReq
		}
	}
}

func handlePreacceptReply(from int64, req *pb.PreacceptReply) error {
	p_out("preacceptReply, slot: %d, len: %d\n", len(logs[me]), req.Slot)
	logInst := logs[me][req.Slot]
	if logInst.State == pb.InstState_COMMITTED || logInst.State == pb.InstState_ACCEPTED || logInst.State == pb.InstState_EXECUTED {
		// committed
		return nil
	}

	if _, ok := replyCount[req.Slot]; !ok {
		replyCount[req.Slot] = 1
	} else {
		replyCount[req.Slot] += 1
	}

	if req.Changed {
		logInst.Seq = int64(max(int(logInst.Seq), int(req.Seq)))
		for i := 0; i < N; i++ {
			if logInst.Deps[i] != req.Deps[i] {
				logInst.Deps[i] = int64(max(int(logInst.Deps[i]), int(req.Deps[i])))
			}
		}
		logInst.Changed = true
	}
	inst := logInst

	// Only the leader receives PreacceptReply
	// The leader already had a record of instance
	// No need to check

	count := replyCount[inst.Slot]
	if count*2+1 >= N {
		replyCount[inst.Slot] = 0
		if logInst.Changed {
			// second phase
			logInst.State = pb.InstState_ACCEPTED
			acceptReq := new(pb.AcceptRequest)
			acceptReq.Inst = logInst
			pReq := wrapAcceptRequest(acceptReq)
			for _, i := range thriftyReps {
				outChannels[i%N] <- pReq
			}
		} else {
			// commit
			committed[req.Slot] = true
			go commit(me, req.Slot)
		}
	}
	return nil
}

func handleAcceptRequest(from int64, req *pb.AcceptRequest) *pb.PeerRequest {

	//p_out("Accept from: %d\n", from)

	inst := req.Inst
	if len(logs[from]) <= int(inst.Slot) {
		logs[from] = append(logs[from], make([]*pb.Instance, int(inst.Slot)+1-len(logs[from]))...)
	}
	logs[from][inst.Slot] = inst
	// acceptReply
	acceptReply := new(pb.AcceptReply)
	acceptReply.Slot = inst.Slot
	pReq := wrapAcceptReply(acceptReply)
	outChannels[from] <- pReq
	return pReq
}

func handleAcceptReply(from int64, req *pb.AcceptReply) error {
	replyCount[req.Slot]++
	if c, ok := committed[req.Slot]; !ok || !c {
		if replyCount[req.Slot]*2+1 >= N {
			committed[req.Slot] = true
			go commit(me, req.Slot)
		}
	}
	return nil
}

func handleCommit(from int64, req *pb.CommitRequest) {
	inst := req.Inst
	if len(logs[inst.Rep]) <= int(inst.Slot) {
		logs[inst.Rep] = append(logs[inst.Rep], make([]*pb.Instance, int(inst.Slot)+1-len(logs[inst.Rep]))...)
	}
	logs[inst.Rep][inst.Slot] = inst
}

//=====================================================================

func p_out(s string, args ...interface{}) {
	if debug > 0 {
		return
	}
	fmt.Printf(strconv.Itoa(me)+":"+s, args...)
}

func p_level(level int, s string, args ...interface{}) {
	if debug >= level {
		fmt.Printf(strconv.Itoa(me)+":"+s, args...)
	}
}

func p_err(s string, args ...interface{}) {
	fmt.Printf(strconv.Itoa(me)+":"+s, args...)
}

func p_bare(s string, args ...interface{}) {
	fmt.Printf(s, args...)
}

func p_assert(pred bool, s string) {
	if !pred {
		p_exit(strconv.Itoa(me)+":"+"\nFAIL ASSERT: %q\n\n", s)
	}
}

func p_exit(s string, args ...interface{}) {
	fmt.Printf(strconv.Itoa(me)+":"+s, args...)
	os.Exit(1)
}

func p_exitif(cond bool, s string, args ...interface{}) {
	if cond {
		p_exit(strconv.Itoa(me)+":"+s, args...)
	}
}
