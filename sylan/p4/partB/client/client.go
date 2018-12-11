package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"818fall18/sylan/p4/partB/pb" /* MODIFY ME */

	. "github.com/mattn/go-getopt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Metrics struct {
	Proposals uint64
	Replies   uint64
}

var metrics Metrics

type ConfigAddr struct {
	Host string
	Port string
}

type ConfigFile struct {
	N        int
	Batching int
	Replicas []ConfigAddr
}

type ScriptAccess struct {
	R int // replica
	T pb.AccessType
	K string
	V string
}

type Script struct {
	N        int
	Accesses []ScriptAccess
}

var (
	N             int
	oneAccess     string
	configFile    = "../config.json"
	conflicts     int
	dataSize      = 16
	debug         = true
	destRep       int
	hostname      string
	identity      string
	maxThreads    = 1000
	numAccesses   uint64
	pauseInterval int = 5
	roundRobin        = false
	scriptFile        = ""
	serverModulo      = 1
	servers       []pb.ReplicaClient
)

//=====================================================================

func main() {
	hostname, _ = os.Hostname()

	for {
		if c := Getopt("a:o:pP:dc:C:n:N:s:t:r"); c == EOF {
			break
		} else {
			switch c {
			case 'a':
				// repID,r|w|p|d,key,value
				oneAccess = OptArg

			case 'd':
				debug = !debug

			case 'c':
				configFile = OptArg

			case 'C':
				conflicts, _ = strconv.Atoi(OptArg)

			case 'n':
				t, _ := strconv.Atoi(OptArg)
				numAccesses = uint64(t)

			case 'N':
				N, _ = strconv.Atoi(OptArg)

			case 'o':
				destRep, _ = strconv.Atoi(OptArg)

			case 'p':
				time.Sleep(time.Duration(pauseInterval) * time.Second)

			case 'P':
				pauseInterval, _ = strconv.Atoi(OptArg)

			case 'r':
				roundRobin = true

			case 's':
				scriptFile = OptArg

			case 't':
				maxThreads, _ = strconv.Atoi(OptArg)

			}
		}
	}

	//=====================================================================

	// Compute the identity
	if hostname != "" {
		identity = fmt.Sprintf("%s-%04X-", hostname, rand.Intn(0x10000))
	} else {
		identity = fmt.Sprintf("%04X-%04X-", rand.Intn(0x10000), rand.Intn(0x10000))
	}

	//=====================================================================
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGTERM, os.Interrupt, os.Kill)
	go func() {
		<-sigch
		writeMetrics()
	}()

	//=====================================================================
	config := new(ConfigFile)
	if dat, err := ioutil.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(dat, config); err != nil {
			p_exit("Unable to unmarshal config: %v\n", err)
		}
	} else {
		p_exit("Unable to open config file: %v\n", err)
	}
	if N == 0 {
		N = config.N
	}

	// Read our script
	var script Script
	if scriptFile != "" {
		if dat, err := ioutil.ReadFile(scriptFile); err == nil {
			if err := json.Unmarshal(dat, &script); err != nil {
				p_exit("Unable to unmarshal script: %v\n", err)
			}
			numAccesses = uint64(len(script.Accesses))
			if script.N > 0 {
				N = script.N
			}
		} else {
			p_exit("Unable to open script file: %v\n", err)
		}
	}

	servers = make([]pb.ReplicaClient, N)
	for i := 0; i < N; i++ {
		addr := config.Replicas[i].Host + ":" + config.Replicas[i].Port
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			panic(fmt.Sprintf("did no t connect: %v", err))
		}
		p_out("Dialed %q\n", addr)
		servers[i] = pb.NewReplicaClient(conn)
	}

	if roundRobin {
		serverModulo = N
	}

	//=====================================================================

	p_out("Ready to go %v accesses, debug=%v, round=%v from %q, %s/%s, %d servers\n",
		numAccesses, debug, serverModulo, hostname, configFile, scriptFile, N)

	//=====================================================================
	// Do it

	// Initialize the keys and values so that it's not part of throughput.

	// Create the wait group for all threads
	var startTime time.Time
	mono := uint64(1)

	if oneAccess != "" {
		flags := strings.Split(oneAccess, ",")
		repID, _ := strconv.Atoi(flags[0])
		key := flags[2]
		value := flags[3]

		var rep *pb.ClientReply
		var err error

		switch flags[1] {
		case "r":
			if rep, err = servers[repID].Get(context.TODO(), &pb.GetRequest{
				Identity: identity,
				Key:      key,
			}); err != nil {
				p_exit("Send to server %v failed: %v\n", repID, err)
			}
			p_err("read %q: %q from %d\n", key, rep.Pair.Value, repID)
		case "w":
			if rep, err = servers[repID].Put(context.TODO(), &pb.PutRequest{
				Identity: identity,
				Key:      key,
				Value:    []byte(value),
			}); err != nil {
				p_exit("Send to server %v failed: %v\n", repID, err)
			}
			p_err("wrote %q: %q to %d\n", key, value, repID)
		case "d":
			if rep, err = servers[repID].Del(context.TODO(), &pb.DelRequest{
				Identity: identity,
				Key:      key,
			}); err != nil {
				p_exit("Send to server %v failed: %v\n", repID, err)
			}
			p_err("deleted %q at %d\n", key, repID)
		case "p":
			time.Sleep(time.Duration(pauseInterval) * time.Second)
			p_err("paused %q\n", key)
		}

	} else if scriptFile != "" {
		startTime = time.Now()
		for _, acc := range script.Accesses {
			var rep *pb.ClientReply
			var err error

			metrics.Proposals++

			switch acc.T {
			case pb.AccessType_ACCESS_WRITE:
				if rep, err = servers[acc.R].Put(context.TODO(), &pb.PutRequest{
					Identity: identity,
					Key:      acc.K,
					Value:    []byte(acc.V),
				}); err != nil {
					p_exit("Send to server %v failed: %v\n", acc.R, err)
				}
			case pb.AccessType_ACCESS_READ:
				if rep, err = servers[acc.R].Get(context.TODO(), &pb.GetRequest{
					Identity: identity,
					Key:      acc.K,
				}); err != nil {
					p_exit("Send to server %v failed: %v\n", acc.R, err)
				}
			case pb.AccessType_ACCESS_DEL:
				if rep, err = servers[acc.R].Del(context.TODO(), &pb.DelRequest{
					Identity: identity,
					Key:      acc.K,
				}); err != nil {
					p_exit("Send to server %v failed: %v\n", acc.R, err)
				}
			case pb.AccessType_ACCESS_PAUSE:
				time.Sleep(time.Duration(pauseInterval) * time.Second)
			}
			p_out("reply %v\n", rep)

			/*			if _, err := servers[acc.R].Propose(context.TODO(), &pb.ProposeRequest{
						Op: &pb.Operation{
							Identity: identity + strconv.Itoa(int(mono)),
							Type:     pb.AccessType(acc.T),
							Key:      acc.K,
							Value:    []byte(acc.V)}}); err != nil {
						p_exit("Send to server %v failed: %v\n", acc.R, err)  */
		}
		mono++
		metrics.Replies++
	} else {
		keys := make([]string, numAccesses)
		vals := make([][]byte, numAccesses)

		for i := uint64(0); i < numAccesses; i++ {
			keys[i] = fmt.Sprintf("%X", i)
			vals[i] = make([]byte, dataSize)
			rand.Read(vals[i])
		}

		for i := uint64(0); i < numAccesses*uint64((float64(conflicts)/100.0)); i++ {
			keys[i] = "XXX"
		}

		// Execute the blast operation against the server, at most maxThreads
		threads := min(numAccesses, uint64(maxThreads))
		stride := numAccesses/uint64(maxThreads) + 1

		group := new(sync.WaitGroup)
		group.Add(int(threads))
		startTime = time.Now()

		for i := uint64(0); i < threads; i++ {
			mono += stride
			metrics.Proposals += stride

			go func(k uint64, mon uint64) {
				for {
					ident := identity + strconv.Itoa(int(mon))
					p_out("Proposing to server %d, ident %q, key %q\n", k%uint64(serverModulo), ident, keys[k])

					where := k % uint64(serverModulo)
					if destRep > 0 {
						where = uint64(destRep)
					}
					if _, err := servers[where].Put(context.TODO(), &pb.PutRequest{
						Identity: identity,
						Key:      keys[k],
						Value:    vals[k],
					}); err != nil {
						p_exit("Send to server %v failed: %v\n", where, err)
					}

					/*					if _, err := servers[k%uint64(serverModulo)].Propose(context.TODO(),
										&pb.ProposeRequest{
											Op: &pb.Operation{
												Identity: ident,
												Type:     pb.AccessType_ACCESS_WRITE,
												Key:      keys[k],
												Value:    vals[k]}}); err != nil {
										p_exit("Send to server %v failed: %v\n", k%uint64(serverModulo), err)
									} */
					metrics.Replies++

					k += threads
					if k >= numAccesses {
						break
					}
					mon++
				}
				group.Done()
			}(i, mono)
		}

		group.Wait()
	}

	endTime := time.Now()
	elapsed := endTime.Sub(startTime)

	//=====================================================================
	ns := elapsed.Nanoseconds() + 1
	p_out("%d accesses in %v nsecs, rate of %v / sec\n", numAccesses, ns, (int64(numAccesses)*1000000000)/ns)

	writeMetrics()
}

func writeMetrics() {
	b, _ := json.MarshalIndent(metrics, "   ", "   ")
	if debug {
		p_err("\n\n%s\n", string(b))
	}
	ioutil.WriteFile(fmt.Sprintf("output.client"), b, 0644)
	os.Exit(1)
}

//=====================================================================

func p_out(s string, args ...interface{}) {
	if !debug {
		return
	}
	fmt.Printf(s, args...)
}

func p_err(s string, args ...interface{}) {
	fmt.Printf(s, args...)
}

func p_exit(s string, args ...interface{}) {
	fmt.Printf(s, args...)
	writeMetrics()
	os.Exit(1)
}

//=====================================================================

func min(a, b uint64) uint64 {
	if a < b {
		return a
	} else {
		return b
	}
}
