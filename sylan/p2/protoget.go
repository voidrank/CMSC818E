//
package main

/*
 Two main files are ../fuse.go and ../fs/serve.go
*/

import (
	"818fall18/sylan/p2/pb"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"

	"bazil.org/fuse"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
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
	db  *leveldb.DB
	err error
)

//=====================================================================

func importProtoHead(bytes []byte) *pb.Head {
	in := new(pb.Head)
	if err := proto.Unmarshal(bytes, in); err != nil {
		return nil
	}
	return in
}

func importProtoNode(bytes []byte, out *DNode) error {
	in := new(pb.Node)
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

func exportProtoNode(in *DNode) ([]byte, error) {
	out := new(pb.Node)

	out.Attrs = new(pb.Attrtype)
	out.Name = in.Name
	out.Version = uint64(in.Version)
	out.PrevSig = in.PrevSig

	out.Attrs.Valid = int64(in.Attrs.Valid)
	out.Attrs.Inode = in.Attrs.Inode
	out.Attrs.Size = in.Attrs.Size
	out.Attrs.Blocks = in.Attrs.Blocks
	out.Attrs.FileMode = uint32(in.Attrs.Mode)
	out.Attrs.Nlink = in.Attrs.Nlink
	out.Attrs.Uid = in.Attrs.Uid
	out.Attrs.Gid = in.Attrs.Gid
	out.Attrs.Rdev = in.Attrs.Rdev
	out.Attrs.Flags = in.Attrs.Flags
	out.Attrs.BlockSize = in.Attrs.BlockSize

	out.Attrs.Mtime = uint64(in.Attrs.Mtime.UnixNano())
	out.Attrs.Atime = uint64(in.Attrs.Atime.UnixNano())
	out.Attrs.Ctime = uint64(in.Attrs.Ctime.UnixNano())
	out.Attrs.Crtime = uint64(in.Attrs.Crtime.UnixNano())

	out.ChildSigs = in.ChildSigs
	out.DataBlocks = in.DataBlocks

	//	p_err("IN %v (%d), OUT %v (%d)\n", in.DataBlocks, len(in.DataBlocks), out.DataBlocks, len(out.DataBlocks))

	return proto.Marshal(out)
}

//=====================================================================

func main() {
	if len(os.Args) < 3 {
		fmt.Println("USAGE: sget <db path> ('list'|'all'|<key>|</path/.../name>)\n")
	} else {
		o := &opt.Options{
			ReadOnly: true,
		}
		db, err = leveldb.OpenFile(os.Args[1], o)
		if err != nil {
			fmt.Printf("Problem opening DB dir %q\n", os.Args[1])
		} else {
			if (os.Args[2] == "all") || (os.Args[2] == "list") {
				iter := db.NewIterator(nil, nil)
				for iter.Next() {
					key := iter.Key()
					value := iter.Value()
					if os.Args[2] == "list" {
						fmt.Printf("%30q (%d bytes)\n", key, len(value))
					} else {
						fmt.Printf("%30q: '%s'\n", key, string(value))
					}
				}
				iter.Release()
			} else if (os.Args[2] != "") && (os.Args[2][:1] == "/") {
				val := os.Args[2]
				h := getHead()
				if h == nil {
					fmt.Print("Head not found.")
					return
				}
				val = strings.TrimRight(val, "/")

				n := getNode([]byte(h.Root))
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
						n2 := getNode([]byte(n.ChildSigs[k]))
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
			} else if os.Args[2] == "head" {
				b, err := db.Get([]byte(os.Args[2]), nil)
				if err != nil {
					return
				}
				if h := importProtoHead(b); h != nil {
					arr, _ := json.MarshalIndent(h, "    ", "    ")
					fmt.Println(string(arr))
				}
			} else {
				if n := getNode([]byte(os.Args[2])); n != nil {
					arr, err := json.MarshalIndent(n, "    ", "    ")
					if err == nil {
						fmt.Println(string(arr))
					}
				} else {
					b, err := db.Get([]byte(os.Args[2]), nil)
					if err != nil {
						return
					}
					fmt.Println(string(b))
				}
			}
		}
	}
}

func getBlock(key string) []byte {
	if value, err := db.Get([]byte(key), nil); err == nil {
		return value
	} else {
		return nil
	}
}

func getHead() *pb.Head {
	b, err := db.Get([]byte("head"), nil)
	if err != nil {
		return nil
	}

	if h := importProtoHead(b); h != nil {
		return h
	}
	return nil
}

func getPathNode(n *DNode, p string) *DNode {
	if p == "" {
		return n
	}
	strs := strings.Split(p, "/")
	n2 := getNode([]byte(n.ChildSigs[strs[0]]))
	if n2 != nil {
		return getPathNode(n2, strings.Join(strs[1:], "/"))
	} else {
		return nil
	}
}

func getNode(b []byte) *DNode {
	if b == nil {
		return nil
	}
	b, err := db.Get(b, nil)
	if err != nil {
		return nil
	}

	n := new(DNode)
	if err = importProtoNode(b, n); err == nil {
		return n
	} else {
		return nil
	}
}
