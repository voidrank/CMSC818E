// https://play.golang.org/p/MMmG3F0GQhE

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"818fall18/sylan/p3/pbl"
	. "github.com/mattn/go-getopt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	debug  = true
	inited = false
	server pbl.LockClient
)

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

func acquire(path string, mode pbl.LockType, port int, rep int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	_, err := server.Acquire(ctx, &pbl.LockAcquireRequest{Path: path, Mode: mode, RepID: int64(rep)})
	if err != nil {
		p_exit("could not acquire: %v", err)
	}
	p_out("gRPC rep %d acquired %q in mode %v\n", rep, path, mode)
	return nil
}

func release(path string, mode pbl.LockType, port int, rep int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := server.Release(ctx, &pbl.LockReleaseRequest{Path: path, Mode: mode, RepID: int64(rep)})
	if err != nil {
		p_exit("could not release: %v", err)
	}
	p_out("gRPC rep %d released %q in mode %v\n", rep, path, mode)
	return nil
}

/*
func releaseAll(port int, rep int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := server.ReleaseAll(ctx, &pbl.LockReleaseAllRequest{RepID: int64(rep)})
	if err != nil {
		p_exit("could not release: %v", err)
	}
	p_out("gRPC rep %d released ALL\n", r.RepID)
	return nil
}
*/

//=====================================================================
func connect(port int) {
	if !inited {
		conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
		if err != nil {
			p_exit("did not connect: %v", err)
		}
		server = pbl.NewLockClient(conn)
		inited = true
	}
}

//=====================================================================
// Paths should start w/ slash, should not end with one unless root dir.

func main() {
	var (
		c    int
		port = 50052
		rep  = 1
	)

	for {
		if c = Getopt("dp:a:A:r:R:i:"); c == EOF {
			break
		}

		switch c {
		case 'p':
			if p, err := strconv.ParseInt(OptArg, 10, 32); err != nil {
				panic("bad port")
			} else {
				port = int(p)
			}
		case 'i':
			if i, err := strconv.ParseInt(OptArg, 10, 32); err != nil {
				panic("bad repID")
			} else {
				rep = int(i)
			}
		case 'd':
			debug = !debug
		case 'a':
			connect(port)
			acquire(OptArg, pbl.LockType_SHARED, port, rep)
		case 'A':
			connect(port)
			acquire(OptArg, pbl.LockType_EXCLUSIVE, port, rep)
		case 'r':
			connect(port)
			release(OptArg, pbl.LockType_SHARED, port, rep)
		case 'R':
			connect(port)
			release(OptArg, pbl.LockType_EXCLUSIVE, port, rep)
		default:
			println("usage: main.go [-d]", c)
			os.Exit(1)
		}
	}
}
