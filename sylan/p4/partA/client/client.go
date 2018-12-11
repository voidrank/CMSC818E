package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"818fall18/sylan/p4/partA/pb" /* MODIFY ME */

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
	T int // pb.AccessType_ACCESS_WRITE, pb.AccessType_ACCESS_READ
	K string
	V string
}

type Script struct {
	N        int
	Accesses []ScriptAccess
}

var (
	debug        = true
	configFile   = "../config.json"
	scriptFile   = ""
	servers      []pb.ReplicaClient
	hostname     string
	pause        int // msec
	roundRobin   = false
	N            int
	conflicts    int
	serverModulo = 1
	numAccesses  uint64
	dataSize     = 16
	maxThreads   = 1000
	identity     string
)

//=====================================================================

func main() {
	hostname, _ = os.Hostname()

	for {
		if c := Getopt("p:dc:C:n:N:s:t:r"); c == EOF {
			break
		} else {
			switch c {
			case 'd':
				debug = !debug

			case 'c':
				configFile = OptArg

			case 'C':
				conflicts, _ = strconv.Atoi(OptArg)

			case 's':
				scriptFile = OptArg

			case 'r':
				roundRobin = true

			case 't':
				maxThreads, _ = strconv.Atoi(OptArg)

			case 'N':
				N, _ = strconv.Atoi(OptArg)

			case 'n':
				t, _ := strconv.Atoi(OptArg)
				numAccesses = uint64(t)

			case 'p':
				pause, _ = strconv.Atoi(OptArg)
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
		b, _ := json.MarshalIndent(metrics, "   ", "   ")
		ioutil.WriteFile(fmt.Sprintf("output.client"), b, 0644)
		p_exit("\n\n%s\n", string(b))
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
		p_err("Dialed %q\n", addr)
		servers[i] = pb.NewReplicaClient(conn)
	}

	if roundRobin {
		serverModulo = N
	}

	//=====================================================================

	p_err("\nReady to go %v accesses, debug=%v, round=%v from %q, %s/%s, %d servers\n\n",
		numAccesses, debug, serverModulo, hostname, configFile, scriptFile, N)

	//=====================================================================
	// Do it

	// Initialize the keys and values so that it's not part of throughput.

	// Create the wait group for all threads
	var startTime time.Time
	mono := uint64(1)

	if scriptFile != "" {
		startTime = time.Now()
		for _, acc := range script.Accesses {
			metrics.Proposals++
			if _, err := servers[acc.R].Propose(context.TODO(), &pb.ProposeRequest{
				Op: &pb.Operation{
					Identity: identity + strconv.Itoa(int(mono)),
					Type:     pb.AccessType(acc.T),
					Key:      acc.K,
					Value:    []byte(acc.V)}}); err != nil {
				p_exit("Send to server %v failed: %v\n", acc.R, err)
			}
			mono++
			metrics.Replies++
			if pause > 0 {
				p_out("pausing\n")
				time.Sleep(time.Duration(pause) * time.Millisecond)
			}
		}
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
					if _, err := servers[k%uint64(serverModulo)].Propose(context.TODO(),
						&pb.ProposeRequest{
							Op: &pb.Operation{
								Identity: ident,
								Type:     pb.AccessType_ACCESS_WRITE,
								Key:      keys[k],
								Value:    vals[k]}}); err != nil {
						p_exit("Send to server %v failed: %v\n", k%uint64(serverModulo), err)
					}
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
	p_err("ns %v\n", ns)
	p_err("\n%d accesses in %v nsecs, rate of %v / sec\n", numAccesses, ns, (int64(numAccesses)*1000000000)/ns)
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
