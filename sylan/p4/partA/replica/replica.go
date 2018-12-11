package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/looplab/tarjan"
	. "github.com/mattn/go-getopt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	//"runtime"
	_ "sort"
	"strconv"
	"strings"
	"syscall"

	//"sync"
	_ "time"

	"818fall18/sylan/p4/partA/pb" /* MODIFY ME */

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
	debug       = true
	aggregating = false
	outfile     = false
	thrifty     = true
	configFile  = "../config.json"
	config      = new(ConfigFile)
	hostname    string

	clients     []pb.ReplicaClient
	outChannels []chan *pb.PeerRequest // out channel to other replicas
	logs        [][]*pb.Instance

	replyCount = make(map[int64]int)
	committed  = make(map[int64]bool)

	proposalChannels = make(map[string]chan *pb.ProposeReply, LOG_MAX) // to proposer
	bigchan          = make(chan *MsgContainer, 1000)

	metrics Metrics
)

//=====================================================================

func main() {
	hostname, _ = os.Hostname()

	metrics.MessageReceives = make(map[string]int64)
	metrics.MessageSends = make(map[string]int64)
	for {
		if c := Getopt("adr:N:oc:t"); c == EOF {
			break
		} else {
			switch c {
			case 'a':
				aggregating = !aggregating

			case 'd':
				debug = !debug

			case 'N':
				N, _ = strconv.Atoi(OptArg)

			case 'o':
				outfile = !outfile

			case 'r':
				me, _ = strconv.Atoi(OptArg)

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
	fmt.Println(config.Replicas[0].Host)
	outChannels = make([]chan *pb.PeerRequest, N)

	clients = make([]pb.ReplicaClient, N)
	for i := 0; i < N; i++ {
		addr := config.Replicas[i].Host + ":" + config.Replicas[i].Port
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			panic(fmt.Sprintf("did no t connect: %v", err))
		}
		p_err("Dialed %q\n", addr)
		clients[i] = pb.NewReplicaClient(conn)
		outChannels[i] = make(chan *pb.PeerRequest)
		go func(k uint64) {
			for req := range outChannels[k] {
				stream, _ := clients[k].Dispatch(context.TODO())
				//p_out("outchannels[%d] receives a msg\n", k)
				stream.Send(req)
			}
		}(uint64(i))
	}

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
	p_err("\nReady... rep %d, port %q, N=%v, agg=%v, reps:%v\n\n", me, config.Replicas[me].Port, N, aggregating, thrifty)
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

// Propose is a unary RPC that allows clients to propose Instances to the quorum.
func (s *server) Propose(ctx context.
	Context, req *pb.ProposeRequest) (*pb.ProposeReply, error) {
	p_out("RECEIVED PROPOSE %q/%q (%q), about to send request to bigchan\n",
		req.Op.Key, req.Op.Value, req.Op.Identity)

	msg := new(MsgContainer)

	msg.clientOp = req.Op
	replyChan := make(chan *pb.ProposeReply)
	msg.clientChan = replyChan
	bigchan <- msg
	reply := <-replyChan

	return reply, nil
}

//=====================================================================

func bigDispatcher() {

	for e := range bigchan {

		// DO NOT MODIFY (I need for testing)
		if e.clientOp != nil {
			p_out("pmsg %q from %v\n", pb.MsgType_name[int32(pb.MsgType_PROPOSE)], e.clientOp.Identity)
			// clientRequest
			handleProposition(e.clientOp)

			proposeReply := new(pb.ProposeReply)
			proposeReply.Slot = int64(gSeq)
			e.clientChan <- proposeReply

		} else {
			p_out("pmsg %q from %v\n", pb.MsgType_name[int32(e.PeerMsg.Type)], e.PeerMsg.From)
			// peerRequest
			pReq := e.PeerMsg
			switch req := (pReq.Message).(type) {
			case *pb.PeerRequest_Preaccept:
				handlePreacceptRequest(pReq.From, req.Preaccept)
			case *pb.PeerRequest_Preacceptreply:
				handlePreacceptReply(pReq.From, req.Preacceptreply)
			case *pb.PeerRequest_Commit:
				handleCommit(pReq.From, req.Commit)
			}
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
			return err
		}
		msg := new(MsgContainer)
		msg.clientOp = nil
		msg.PeerMsg = req
		// encapsulate message and stuff down bigChannel
		bigchan <- msg
	}
}

// in bigchan, so no mutex
func handleProposition(op *pb.Operation) {
	gSeq++
	for i := 0; i < N; i++ {
		if i != me {
			preacceptReq := new(pb.PreacceptRequest)
			inst := new(pb.Instance)
			inst.Slot = int64(gSeq)
			inst.State = pb.InstState_PREACCEPTED
			preacceptReq.Inst = inst
			peerRequest := wrapPreacceptRequest(preacceptReq)
			//p_out("send to channels %d\n", i)
			outChannels[i] <- peerRequest
		}
	}
}

func handlePreacceptRequest(from int64, req *pb.PreacceptRequest) *pb.PeerRequest {
	req_inst := req.Inst
	reply := new(pb.PreacceptReply)
	reply.Slot = req_inst.Slot
	peerRequest := wrapPreacceptReply(reply)
	outChannels[from] <- peerRequest
	return peerRequest
}

func handlePreacceptReply(from int64, req *pb.PreacceptReply) error {
	slot := req.Slot
	if _, ok := committed[slot]; ok {
		// committed
		return nil
	}
	if _, ok := replyCount[slot]; !ok {
		replyCount[slot] = 1
	} else {
		replyCount[slot] += 1
	}

	count := replyCount[slot]
	if count*2+1 >= N {
		committed[slot] = true
		for i := 0; i < N; i++ {
			if i != me {
				slot := req.Slot
				commitReq := new(pb.CommitRequest)
				inst := new(pb.Instance)
				inst.Rep = int64(me)
				inst.Slot = slot
				inst.State = pb.InstState_EXECUTED
				commitReq.Inst = inst
				pReq := wrapCommitRequest(commitReq)
				outChannels[i] <- pReq
			}
		}
	}
	return nil
}

func handleAcceptRequest(from int64, req *pb.AcceptRequest) *pb.PeerRequest {

	return nil
}

func handleAcceptReply(from int64, req *pb.AcceptReply) error {

	return nil
}

func handleCommit(from int64, req *pb.CommitRequest) {

}

//=====================================================================

func p_out(s string, args ...interface{}) {
	if !debug {
		return
	}
	fmt.Printf(strconv.Itoa(me)+":"+s, args...)
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
