// memfs implements a simple in-memory file system.
package main

/*
 Two main files are ../fuse.go and ../fs/serve.go
*/

import (
	"fmt"
	. "github.com/mattn/go-getopt"
	"log"
	"os"
	"os/signal"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	"golang.org/x/net/context"
)

/*
    Need to implement these types from bazil/fuse/fs!

    type FS interface {
	  // Root is called to obtain the Node for the file system root.
	  Root() (Node, error)
    }

    type Node interface {
	  // Attr fills attr with the standard metadata for the node.
	  Attr(ctx context.Context, attr *fuse.Attr) error
    }
*/

//=============================================================================

type Dfs struct{}

type DNode struct {
	nid   uint64
	name  string
	attr  fuse.Attr
	dirty bool
	kids  map[string]*DNode
	data  []uint8
}

var root *DNode
var nextInd uint64
var nodeMap = make(map[uint64]*DNode) // not currently queried
var debug = false
var mountPoint = "dss"
var uid = os.Geteuid()
var gid = os.Getegid()

// We are defining two main structures that match bazil interfaces. Check them here.
var _ fs.Node = (*DNode)(nil)
var _ fs.FS = (*Dfs)(nil)

//=============================================================================

func p_out(s string, args ...interface{}) {
	if !debug {
		return
	}
	fmt.Printf(s, args...)
}

func p_err(s string, args ...interface{}) {
	fmt.Printf(s, args...)
	os.Exit(1)
}

//=============================================================================

func (Dfs) Root() (n fs.Node, err error) {
	return root, nil
}

//=============================================================================

// Helper function.
func (n *DNode) isDir() bool {
	return n.attr.Mode.IsDir()
}

// Helper function.
func (n *DNode) fuseType() fuse.DirentType {
	if n.isDir() {
		// directory
		return fuse.DT_Dir
	} else if n.attr.Mode&os.ModeSymlink != 0 {
		// link
		return fuse.DT_Link
	} else {
		// file
		return fuse.DT_File
	}
	// TODO: Other types
}

// Helper function.
func (n *DNode) init(name string, mode os.FileMode) {
	curTime := time.Now()
	n.nid = nextInd

	// increase nextInd
	nextInd = nextInd + 1
	n.data = make([]uint8, 0)
	n.name = name

	// set attr for the new node
	n.attr.Inode = n.nid
	n.attr.Size = 0
	//n.attr.Blocks = ?
	n.attr.Atime = curTime
	n.attr.Mtime = curTime
	n.attr.Ctime = curTime
	n.attr.Crtime = curTime
	n.attr.Mode = mode
	// n.attr.Nlink = ?
	n.attr.Uid = uint32(uid)
	n.attr.Gid = uint32(gid)
	// n.attr.Rdev = ?
	// n.attr.Flags = ?
	// n.attr.BlockSize = ?

	// no new data
	n.dirty = false

	// if it is dir
	if n.attr.Mode.IsDir() {
		n.kids = make(map[string]*DNode)
	}
}

func (n *DNode) Attr(ctx context.Context, attr *fuse.Attr) error {
	*attr = n.attr
	return nil
}

func (n *DNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if n.kids != nil {
		if kid := n.kids[name]; kid != nil {
			return kid, nil
		}
	}
	// File or Dir Not Found!
	return nil, fuse.ENOENT
}

func (n *DNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// for debug:
	p_out("enter ReadDirAll\n")

	// build the directory list
	retDirList := make([]fuse.Dirent, 0, len(n.kids))
	// append each dir into the list
	for _, filenode := range n.kids {
		dirent := new(fuse.Dirent)
		dirent.Type = filenode.fuseType()
		dirent.Name = filenode.name
		dirent.Inode = filenode.nid
		retDirList = append(retDirList, *dirent)
		p_out("INode: %d, fuseType: %d, Name: %d\n", retDirList[0].Inode, retDirList[0].Type, retDirList[0].Name)
	}
	return retDirList, nil
}

func (n *DNode) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	// for debug:
	//p_out("enter Getattr\n")
	return n.Attr(ctx, &resp.Attr)
}

// must be defined or editing w/ vi or emacs fails. Doesn't have to do anything
func (n *DNode) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}

func (n *DNode) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	// for debug
	p_out("enter Setattr\n")

	// set every propertie of Attr
	if req.Valid.Mode() {
		n.attr.Mode = req.Mode
	}

	if req.Valid.Atime() {
		n.attr.Atime = req.Atime
	}

	if req.Valid.Mtime() {
		n.attr.Mtime = req.Mtime
	}

	if req.Valid.Uid() {
		n.attr.Uid = req.Uid
	}

	if req.Valid.Gid() {
		n.attr.Gid = req.Gid
	}

	if req.Valid.Flags() {
		n.attr.Flags = req.Flags
	}

	if req.Valid.Size() {
		if req.Size < uint64(len(n.data)) {
			n.data = n.data[:req.Size]
		}
		n.attr.Size = req.Size
		p_out("enter here")
	}

	resp.Attr = n.attr
	return nil
}

func (p *DNode) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	// for degug:
	p_out("enter Mkdir")

	new_dir := new(DNode)
	new_dir.init(req.Name, os.ModeDir|req.Mode)
	// if a file with the same name exists
	if p.kids[req.Name] != nil {
		return nil, fuse.EEXIST
	} else {
		// work correctly
		p.kids[req.Name] = new_dir
		return new_dir, nil
	}
}

func (p *DNode) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	// for debug:
	p_out("enter Create\n")

	new_file := new(DNode)
	new_file.init(req.Name, req.Mode)
	//TODO:
	//resp.LookupResponse ???
	// TODO:
	//resp.OpenResponse ???
	if p.kids[req.Name] != nil {
		return nil, nil, fuse.EEXIST
	} else {
		p.kids[req.Name] = new_file
		return new_file, new_file, nil
	}
}

func (n *DNode) ReadAll(ctx context.Context) ([]byte, error) {
	// for debug
	p_out("enter Read All\n")
	return []byte(n.data), nil
}

func (n *DNode) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	// for degug:
	p_out("enter Write\n")

	if int(req.Offset)+len(req.Data) > len(n.data) {
		for i := req.Offset; i < int64(len(n.data)); i++ {
			n.data[i] = req.Data[i-req.Offset]
		}
		n.data = append(n.data, req.Data[int64(len(n.data))-req.Offset:]...)
	} else {
		for i := req.Offset; i < req.Offset+int64(len(req.Data)); i++ {
			n.data[i] = req.Data[i-req.Offset]
		}
		//n.data = n.data[:req.Offset+int64(len(req.Data))]
	}
	n.attr.Size = uint64(len(n.data))
	//n.dirty = true

	resp.Size = len(req.Data)

	return nil
	// TODO: Flags, LockOwner, FileFlags
}

func (n *DNode) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	p_out("flush [%v] %q (dirty: %t, now %d bytes)\n", req, n.name, n.dirty, len(n.data))

	//TODO
	//n.dirty = false
	return nil
}

// hard links. Note that 'name' is not modified, so potential debugging problem.
func (p *DNode) Link(ctx context.Context, req *fuse.LinkRequest, oldNode fs.Node) (fs.Node, error) {
	p_out("enter Link\n")
	curDNode := oldNode.(*DNode)
	// Hard link of dir is not allowed
	if curDNode.isDir() {
		return nil, fuse.EPERM
	} else {
		curTime := time.Now()
		curDNode.attr.Atime = curTime
		curDNode.attr.Mtime = curTime
		p.kids[req.NewName] = curDNode
		return curDNode, nil
	}
}

func (n *DNode) Remove(ctx context.Context, req *fuse.RemoveRequest) error {

	p_out("enter Remove\n")
	if _, res := n.kids[req.Name]; !res {
		// no such file or directory
		return fuse.ENOENT
	} else {
		// rm dir without req.Dir == 1
		if n.kids[req.Name].isDir() && !req.Dir {
			return fuse.EPERM
		} else {
			delete(n.kids, req.Name)
			return nil
		}
	}
	return nil
}

func (p *DNode) Symlink(ctx context.Context, req *fuse.SymlinkRequest) (fs.Node, error) {
	p_out("Sym Link Call   -- receiver id %d, new name: %q, target name: %q\n", p.nid, req.NewName, req.Target)
	d := new(DNode)
	d.init(req.NewName, os.ModeSymlink|0777)
	p.kids[req.NewName] = d
	d.data = []byte(req.Target)
	d.dirty = true
	d.attr.Size = uint64(len(d.data))
	return d, nil
}

func (n *DNode) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	// TODO:
	return "", nil
}

func (n *DNode) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	p_out("enter Rename\n")

	if _, res := n.kids[req.OldName]; !res {
		// file not found
		return fuse.ENOENT
	} else {
		newDNode := newDir.(*DNode)
		newDNode.kids[req.NewName] = n.kids[req.OldName]
		newDNode.kids[req.NewName].name = req.NewName
		delete(n.kids, req.OldName)
		return nil
	}

}

// func (n fs.Node) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error){}
// func (n fs.Node) Release(ctx context.Context, req *fuse.ReleaseRequest) error {}
// func (n fs.Node) Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error {}
// func (n fs.Node) Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error {}
// func (n fs.Node) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {}
// func (n fs.Node) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {}

//=============================================================================

func main() {
	var c int

	for {
		if c = Getopt("dm:"); c == EOF {
			break
		}

		switch c {
		case 'd':
			debug = !debug
		case 'm':
			mountPoint = OptArg
		default:
			println("usage: main.go [-d | -m <mountpt>]", c)
			os.Exit(1)
		}
	}

	p_out("main\n")

	root = new(DNode)
	root.init("", os.ModeDir|0755)

	nodeMap[uint64(root.attr.Inode)] = root
	p_out("root inode %d", int(root.attr.Inode))

	if _, err := os.Stat(mountPoint); err != nil {
		os.Mkdir(mountPoint, 0755)
	}
	fuse.Unmount(mountPoint)
	conn, err := fuse.Mount(mountPoint, fuse.FSName("dssFS"), fuse.Subtype("project P1"),
		fuse.LocalVolume(), fuse.VolumeName("dssFS"))
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go func() {
		<-ch
		defer conn.Close()
		fuse.Unmount(mountPoint)
		os.Exit(1)
	}()

	err = fs.Serve(conn, Dfs{})
	p_out("AFTER\n")
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-conn.Ready
	if err := conn.MountError; err != nil {
		log.Fatal(err)
	}
}
