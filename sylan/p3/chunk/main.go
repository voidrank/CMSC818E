package main

import (
	"log"
	"net"
	"os"

	pb "818fall18/sylan/p3/pbc"
	. "github.com/mattn/go-getopt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	PORT = ":50051"
)

type server struct{}

type Stats struct {
	Sends    int
	Receives int
	BytesIn  int
	BytesOut int
}

var stats Stats

var (
	debug     = false
	db        *leveldb.DB
	cacheSize = 8 * 1048576
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

//=====================================================================

// your stuff
func (s *server) Put(ctx context.Context, in *pb.ChunkPutRequest) (*pb.ChunkPutReply, error) {
	key := []byte(in.Key)
	value := in.Value

	err := db.Put(key, value, nil)
	if err == nil {
		return &pb.ChunkPutReply{}, nil
	} else {
		return &pb.ChunkPutReply{ErrorString: "error"}, err
	}
}

func (s *server) Get(ctx context.Context, in *pb.ChunkGetRequest) (*pb.ChunkGetReply, error) {
	key := []byte(in.Key)
	value, err := db.Get(key, nil)
	if err == nil {
		return &pb.ChunkGetReply{Key: in.Key, Value: value}, nil
	} else {
		return nil, err
	}
}

func (s *server) List(ctx context.Context, in *pb.ChunkListRequest) (*pb.ChunkListReply, error) {
	iter := db.NewIterator(nil, nil)
	count := 0
	for iter.Next() {
		count += 1
	}
	returnList := make([]string, count)
	cur_idx := 0
	for iter.Next() {
		key := iter.Key()
		returnList[cur_idx] = string(key)
	}
	return &pb.ChunkListReply{Keys: returnList}, nil
}

//=====================================================================

func main() {
	var (
		c         int
		newfs     = false
		storePath = "db"
		o         *opt.Options
		status    = ""
	)

	for {
		if c = Getopt("nds:"); c == EOF {
			break
		}

		switch c {
		case 'd':
			debug = !debug
		case 'n':
			newfs = !newfs
		case 's':
			storePath = OptArg
		default:
			println("usage: main.go [-d | -s <db path>]", c)
			os.Exit(1)
		}
	}
	if newfs {
		status += " NEWFS"
	}
	if debug {
		status += " DEBUG"
	}

	p_err("ChunkServer%s at %q\n", status, storePath)

	if newfs {
		os.RemoveAll(storePath)
	}

	if cacheSize > 0 {
		o = &opt.Options{
			BlockCacheCapacity: 100 * 1048576,
		}
	}

	var err error
	db, err = leveldb.OpenFile(storePath, o)
	if err != nil {
		panic("no open db\n")
	}

	// REGISTER CHUNKSERVER
	lis, err := net.Listen("tcp", PORT)
	if err != nil {
		p_out("failed to listen: %v", err)
	} else {
		s := grpc.NewServer()
		pb.RegisterChunkServer(s, &server{})
		reflection.Register(s)
		if err := s.Serve(lis); err != nil {
			p_out("failed to server: %v", err)
		}
	}
}
