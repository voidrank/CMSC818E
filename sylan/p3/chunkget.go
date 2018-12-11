//
package main

/*
 Two main files are ../fuse.go and ../fs/serve.go
*/

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	. "github.com/mattn/go-getopt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"

	"818fall18/sylan/p3/pbc"
	"bazil.org/fuse"
	"os"
	"sort"
	"strings"
	"time"
)

type DNode struct {
	Name       string
	Attrs      fuse.Attr
	Version    int
	PrevSig    string
	ChildSigs  map[string]string
	DataBlocks []string
}

var (
	err   error
	debug = false
)

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
func importProtoHead(bytes []byte) *pbc.Head {
	in := new(pbc.Head)
	if err := proto.Unmarshal(bytes, in); err != nil {
		return nil
	}
	return in
}

func importProtoNode(bytes []byte, out *DNode) error {
	in := new(pbc.Node)
	if err := proto.Unmarshal(bytes, in); err != nil {
		return err
	}

	out.Name = in.Name
	out.Version = int(in.Version)
	out.PrevSig = in.PrevSig

	out.Attrs.Valid = time.Duration(in.Attrs.Valid)
	out.Attrs.Inode = in.Attrs.Inode
	out.Attrs.Size = in.Attrs.Size
	out.Attrs.Blocks = in.Attrs.Blocks
	out.Attrs.Mode = os.FileMode(in.Attrs.FileMode)
	out.Attrs.Nlink = in.Attrs.Nlink
	out.Attrs.Uid = in.Attrs.Uid
	out.Attrs.Gid = in.Attrs.Gid
	out.Attrs.Rdev = in.Attrs.Rdev
	out.Attrs.Flags = in.Attrs.Flags
	out.Attrs.BlockSize = in.Attrs.BlockSize

	out.Attrs.Mtime = time.Unix(0, int64(in.Attrs.Mtime))
	out.Attrs.Ctime = time.Unix(0, int64(in.Attrs.Ctime))
	out.Attrs.Crtime = time.Unix(0, int64(in.Attrs.Crtime))
	out.Attrs.Atime = time.Unix(0, int64(in.Attrs.Atime))

	out.ChildSigs = in.ChildSigs
	out.DataBlocks = in.DataBlocks

	return nil
}

//=====================================================================

func main() {
	var c int
	address := "localhost:50051"

	for {
		if c = Getopt("a:d"); c == EOF {
			break
		}

		switch c {
		case 'd':
			debug = !debug
		case 'a':
			address = OptArg
		default:
			println("usage: main.go [-d | -a <network addr>] <key>", c)
			os.Exit(1)
		}
	}
	if OptInd >= len(os.Args) {
		println("usage: main.go [-d | -a <network addr>] <key>", c)
		os.Exit(1)
	}
	key := os.Args[OptInd]
	p_err("\nStarting up w/ address %q, key %q\n", address, key)

	initRPC(address)
	if (key == "all") || (key == "list") {
		for _, k := range chunkList() {
			p_err("\t%q\n", k)
		}
	} else if (key != "") && (key[:1] == "/") {
		val := key
		h := getHead()
		if h == nil {
			return
		}
		val = strings.TrimRight(val, "/")

		n := getNode(h.Root)
		if n == nil {
			return
		}
		val = strings.TrimLeft(val, "/")
		n = getPathNode(n, val)
		if n == nil {
			return
		}

		if len(n.ChildSigs) > 0 {
			mk := make([]string, len(n.ChildSigs))
			i := 0
			for k, _ := range n.ChildSigs {
				mk[i] = k
				i++
			}
			sort.Strings(mk)

			for _, k := range mk {
				dir := ""
				n2 := getNode(n.ChildSigs[k])
				if (int(n2.Attrs.Mode) & int(os.ModeDir)) != 0 {
					dir = "/"
				}
				if val == "" {
					fmt.Printf("%q  %s%s\n", n.ChildSigs[k], k, dir)
				} else {
					fmt.Printf("%q  %s%s\n", n.ChildSigs[k], k, dir)
				}
			}
		} else {
			arr, err := json.MarshalIndent(n, "    ", "    ")
			if err == nil {
				fmt.Println(string(arr))
				lens := []int{}
				for _, b := range n.DataBlocks {
					lens = append(lens, len(getBlock(b)))
				}
				fmt.Printf("DataBlock lens: %v\n", lens)
			}
		}
	} else if key == "head" {
		if h := getHead(); h != nil {
			arr, _ := json.MarshalIndent(h, "    ", "    ")
			fmt.Println(string(arr))
		}
	} else {
		if n := getNode(key); n != nil {
			arr, err := json.MarshalIndent(n, "    ", "    ")
			if err == nil {
				fmt.Println(string(arr))
			}
		} else {
			if b, err := chunkGet(key); err != nil {
				return
			} else {
				fmt.Println(string(b))
			}
		}
	}
}

func getBlock(key string) []byte {
	if value, err := chunkGet(key); err == nil {
		return value
	} else {
		return nil
	}
}

func getHead() *pbc.Head {
	if b, err := chunkGet("head"); err != nil {
		return nil
	} else {
		if h := importProtoHead(b); h != nil {
			return h
		}
		return nil
	}
}

func getPathNode(n *DNode, p string) *DNode {
	if p == "" {
		return n
	}
	strs := strings.Split(p, "/")
	n2 := getNode(n.ChildSigs[strs[0]])
	if n2 != nil {
		return getPathNode(n2, strings.Join(strs[1:], "/"))
	} else {
		return nil
	}
}

func getNode(b string) *DNode {
	if b == "" {
		return nil
	}
	bytes, err := chunkGet(b)
	if err != nil {
		return nil
	}

	n := new(DNode)
	if err = importProtoNode(bytes, n); err == nil {
		return n
	} else {
		return nil
	}
}

//=====================================================================

var (
	server pbc.ChunkClient
)

func initRPC(address string) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	server = pbc.NewChunkClient(conn)
}

func chunkPut(key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := server.Put(ctx, &pbc.ChunkPutRequest{Key: key, Value: value})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	p_out("gRPC PUT %d bytes of %q\n", len(value), key)
	return nil
}

func chunkGet(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	reply, err := server.Get(ctx, &pbc.ChunkGetRequest{Key: key})
	if err != nil {
		log.Fatalf("could not GET: %v", err)
	}
	p_out("gRPC GET %d bytes of %q\n", len(reply.Value), key)
	return reply.Value, nil
}

func chunkList() []string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if reply, err := server.List(ctx, &pbc.ChunkListRequest{}); err != nil {
		p_exit("could not LIST: %v", err)
		return []string{}
	} else {
		p_out("gRPC LIST %d keys\n", len(reply.Keys))
		return reply.Keys
	}
}
