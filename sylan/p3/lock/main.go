// https://play.golang.org/p/MMmG3F0GQhE

package main

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"

	pb "818fall18/sylan/p3/pbl"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	PORT = ":50052"
)

type serverObj struct{}

type LockNotExistError struct{}
type NotAPathError struct{}

func (e LockNotExistError) Error() string {
	return "Lock doesn't exist"
}

func (e NotAPathError) Error() string {
	return "Invalid Path"
}

var (
	debug  = true
	mLock  = make(map[int]map[string]*sync.RWMutex)
	tLock  = make(map[int]map[string]*sync.RWMutex)
	cLock  = &sync.Mutex{}
	server = &serverObj{}
)

func p_out(s string, args ...interface{}) {
	if !debug {
		return
	}
	log.Printf(s, args...)
}

func p_err(s string, args ...interface{}) {
	log.Printf(s, args...)
}

func p_exit(s string, args ...interface{}) {
	log.Printf(s, args...)
	os.Exit(1)
}

func filterPath(in string) (string, error) {
	if len(in) <= 0 {
		return "", new(NotAPathError)
	} else {
		if in[0] != '/' {
			return "", new(NotAPathError)
		} else {
			if in[len(in)-1] == '/' {
				out := in[:len(in)-1]
				return out, nil
			} else {
				return in, nil
			}
		}
	}
}

//=====================================================================

func (s *serverObj) Acquire(ctx context.Context, in *pb.LockAcquireRequest) (*pb.LockAcquireReply, error) {
	_path := in.Path
	mode := in.Mode
	repID := int(in.RepID)
	accPath := ""

	if path, err := filterPath(_path); err != nil {
		return nil, err
	} else {
		if _, ok := mLock[repID]; !ok {
			mLock[repID] = make(map[string]*sync.RWMutex)
			tLock[repID] = make(map[string]*sync.RWMutex)
		}

		if _, ok := tLock[repID][path]; !ok {
			tLock[repID][path] = &sync.RWMutex{}
		}
		if mode == pb.LockType_SHARED {
			tLock[repID][path].RLock()
		} else {
			tLock[repID][path].Lock()
		}

		for _, name := range strings.Split(path, "/") {
			if len(accPath) == 1 {
				accPath = accPath + name
			} else {
				accPath = accPath + "/" + name
			}
			if _, ok := mLock[repID][accPath]; !ok {
				cLock.Lock()
				if _, ok := mLock[repID][accPath]; !ok {
					mLock[repID][accPath] = &sync.RWMutex{}
				}
				cLock.Unlock()
			}
			lock := mLock[repID][accPath]
			if mode == pb.LockType_SHARED || accPath != path {
				lock.RLock()
				p_out("repID: %d, rlock %s", repID, accPath)
			} else {
				lock.Lock()
				p_out("repID: %d, lock %s", repID, accPath)
			}
		}
		p_out("finish.")

		return &pb.LockAcquireReply{Path: path, Mode: mode, RepID: int64(repID)}, nil
	}
}

func (s *serverObj) Release(ctx context.Context, in *pb.LockReleaseRequest) (*pb.LockReleaseReply, error) {
	_path := in.Path
	mode := in.Mode
	repID := int(in.RepID)
	accPath := ""

	if path, err := filterPath(_path); err != nil {
		return nil, err
	} else {
		if _, ok := tLock[repID]; !ok {
			return nil, new(LockNotExistError)
		} else {
			if tMutex, ok := tLock[repID][path]; !ok {
				return nil, new(LockNotExistError)
			} else {
				if mode == pb.LockType_SHARED {
					tMutex.RUnlock()
				} else {
					tMutex.Unlock()
				}
				accPath = ""
				for _, name := range strings.Split(path, "/") {
					if len(accPath) == 1 {
						accPath = accPath + name
					} else {
						accPath = accPath + "/" + name
					}
					lock := mLock[repID][accPath]
					if mode == pb.LockType_SHARED || accPath != path {
						lock.RUnlock()
					} else {
						lock.Unlock()
					}
				}
				return &pb.LockReleaseReply{Path: path, Mode: mode, RepID: int64(repID)}, nil
			}
		}
	}

}

func main() {

	lis, err := net.Listen("tcp", PORT)
	if err != nil {
		p_out("failed to listen: %v", err)
	} else {
		s := grpc.NewServer()
		pb.RegisterLockServer(s, server)
		reflection.Register(s)
		if err := s.Serve(lis); err != nil {
			p_out("failed to serve: %v", err)
		}
	}
}
