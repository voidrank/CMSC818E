//
package dfs

/*

 Two main files are ../fuse.go and ../fs/serve.go
*/

import (
	//"dss/p2/solution/pb"
	"818fall18/sylan/p2/pb"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	//"encoding/json"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"golang.org/x/net/context"
	"log"
	//"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	//"sync/atomic"
	"time"
)

type DNode struct {
	Name    string
	Attrs   fuse.Attr
	Version int
	PrevSig string

	ChildSigs map[string]string

	DataBlocks []string

	sig        string
	dirty      bool
	metaDirty  bool
	expanded   bool
	loadFromDB bool
	parent     *DNode
	kids       map[string]*DNode
	data       []byte
	mutex      sync.Mutex
}

type FS struct{}

var _ fs.Node = (*DNode)(nil)
var _ fs.FS = (*FS)(nil)

var (
	flushMutex sync.Mutex
	err        error
	debug      bool
	useProto   = true
	compress   bool
	readOnly   bool
	nodeMap    = make(map[string]*DNode)
	db         *leveldb.DB
	root       *DNode
	nextInd    uint64
	uid        = os.Geteuid()
	gid        = os.Getegid()
)

//=============================================================================
// Let one at a time in

func Lock() {
	flushMutex.Lock()
}

func Unlock() {
	flushMutex.Unlock()
}

func (FS) Root() (n fs.Node, err error) {
	p_out("FS Root, Inode: %d", root.Attrs.Inode)
	return root, nil
}

func (n *DNode) Lock() {
	n.mutex.Lock()
}

func (n *DNode) Unlock() {
	n.mutex.Unlock()
}

func (p *DNode) SHA1() (str string, err error) {
	h := sha1.New()
	if bytes, err := exportProtoNode(p); err != nil {
		p_out("OHHHHHHHHHHHHHHHHH NOOOOOOO")
		p_out(err.Error())
		p_out(p.PrevSig)
		return "", err
	} else {
		h.Write(bytes)
		sha1 := h.Sum(nil)
		sha1str := base64.StdEncoding.EncodeToString(sha1)
		//p_out("SHA: %s", sha1str[:10])
		return sha1str, nil
	}
}

// Helper function.
func (n *DNode) isDir() bool {
	return n.Attrs.Mode.IsDir()
}

// Helper function.
func (n *DNode) fuseType() fuse.DirentType {
	if n.isDir() {
		// directory
		return fuse.DT_Dir
	} else if n.Attrs.Mode&os.ModeSymlink != 0 {
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
	n.Attrs.Inode = nextInd
	n.PrevSig = ""

	// increase nextInd
	n.data = make([]uint8, 0)
	n.Name = name

	// set attr for the new node
	n.Attrs.Inode = nextInd
	nextInd = nextInd + 1
	n.Attrs.Size = 0
	//n.attr.Blocks = ?
	n.Attrs.Atime = curTime
	n.Attrs.Mtime = curTime
	n.Attrs.Ctime = curTime
	n.Attrs.Crtime = curTime
	n.Attrs.Mode = mode
	// n.attr.Nlink = ?
	n.Attrs.Uid = uint32(uid)
	n.Attrs.Gid = uint32(gid)
	// n.attr.Rdev = ?
	// n.attr.Flags = ?
	// n.attr.BlockSize = ?

	// no new data
	n.dirty = false

	// if it is dir
	if n.Attrs.Mode.IsDir() {
		n.kids = make(map[string]*DNode)
	}

}

func (n *DNode) Attr(ctx context.Context, attr *fuse.Attr) error {
	*attr = n.Attrs
	return nil
}

func (n *DNode) Lookup(ctx context.Context, name string) (fs.Node, error) {

	p_out("Lookup Inode: %d, Name: %s, look for: %s", n.Attrs.Inode, n.Name, name)

	if n.loadFromDB {
		n.loadAllData()
	}
	if n.kids != nil {
		if kid, _ := n.kids[name]; kid != nil {
			return kid, nil
		}
	}
	return nil, fuse.ENOENT
}

func (n *DNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// for debug:
	p_out("ReadDirAll\n")
	if n.loadFromDB {
		n.loadAllData()
	}
	// build the directory list
	retDirList := make([]fuse.Dirent, 0, len(n.kids))
	// append each dir into the list
	for _, filenode := range n.kids {
		dirent := new(fuse.Dirent)
		dirent.Type = filenode.fuseType()
		dirent.Name = filenode.Name
		dirent.Inode = filenode.Attrs.Inode
		retDirList = append(retDirList, *dirent)
		p_out("INode: %d, fuseType: %d, Name: %d\n", retDirList[0].Inode, retDirList[0].Type, retDirList[0].Name)
	}
	return retDirList, nil
}

func (n *DNode) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	// for debug:
	//p_out("enter Getattr\n")
	if n.loadFromDB {
		n.loadAllData()
	}
	return n.Attr(ctx, &resp.Attr)
}

// must be defined or editing w/ vi or emacs fails. Doesn't have to do anything
func (n *DNode) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}

func (n *DNode) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	// for debug
	p_out("enter Setattr\n")

	Lock()
	if n.loadFromDB {
		n.loadAllData()
	}
	// set every propertie of Attr
	if req.Valid.Mode() {
		n.Attrs.Mode = req.Mode
	}

	if req.Valid.Atime() {
		n.Attrs.Atime = req.Atime
	}

	if req.Valid.Mtime() {
		n.Attrs.Mtime = req.Mtime
	}

	if req.Valid.Uid() {
		n.Attrs.Uid = req.Uid
	}

	if req.Valid.Gid() {
		n.Attrs.Gid = req.Gid
	}

	if req.Valid.Flags() {
		n.Attrs.Flags = req.Flags
	}

	if req.Valid.Size() {
		if req.Size < uint64(len(n.data)) {
			n.data = n.data[:req.Size]
		}
		n.Attrs.Size = req.Size
		p_out("enter here")
	}

	update(n)
	Unlock()
	resp.Attr = n.Attrs
	return nil
}

func (p *DNode) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	// for degug:
	if readOnly {
		return nil, fuse.EPERM
	} else {
		p_out("enter Mkdir")

		newDir := new(DNode)
		newDir.init(req.Name, os.ModeDir|req.Mode)
		// if a file with the same name exists
		if p.kids[req.Name] != nil {
			return nil, fuse.EEXIST
		} else {
			// work correctly
			Lock()
			p.kids[req.Name] = newDir
			newDir.parent = p
			update(newDir)
			Unlock()
			return newDir, nil
		}
	}
}

func (p *DNode) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	// for debug:
	if readOnly {
		return nil, nil, fuse.EPERM
	} else {
		newFile := new(DNode)
		newFile.init(req.Name, req.Mode)
		p_out("Create, inode: %d, name: %s\n", newFile.Attrs.Inode, req.Name)
		//TODO:
		//resp.LookupResponse ???
		// TODO:
		//resp.OpenResponse ???
		if p.kids[req.Name] != nil {
			return nil, nil, fuse.EEXIST
		} else {
			Lock()
			p.kids[req.Name] = newFile
			newFile.parent = p
			update(newFile)
			Unlock()
			return newFile, newFile, nil
		}
	}
}

func (n *DNode) loadAllData() {
	n.kids = make(map[string]*DNode)
	for kidName, kidSig := range n.ChildSigs {
		kid := getNode([]byte(kidSig))
		n.kids[kidName] = kid
		kid.parent = n
	}
	n.data = make([]byte, 0)
	for _, dataBlockSig := range n.DataBlocks {
		dataBlock, _ := getData(dataBlockSig)
		n.data = append(n.data, dataBlock...)
	}
	n.loadFromDB = false
}

func (n *DNode) ReadAll(ctx context.Context) ([]byte, error) {
	// for debug
	p_out("Read All, inode : %d\n", n.Attrs.Inode)
	if n.loadFromDB {
		Lock()
		n.loadAllData()
		Unlock()
	}

	return []byte(n.data), nil
}

func (n *DNode) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	// for degug:
	if readOnly {
		return fuse.EPERM
	} else {
		p_out("Write Inode: %d\n", n.Attrs.Inode)

		Lock()
		if int(req.Offset)+len(req.Data) > len(n.data) {
			for i := req.Offset; i < int64(len(n.data)); i++ {
				n.data[i] = req.Data[i-req.Offset]
			}
			n.data = append(n.data, req.Data[int64(len(n.data))-req.Offset:]...)
		} else {
			for i := req.Offset; i < req.Offset+int64(len(req.Data)); i++ {
				n.data[i] = req.Data[i-req.Offset]
			}
		}
		n.Attrs.Size = uint64(len(n.data))
		n.dirty = true
		Unlock()

		resp.Size = len(req.Data)
		return nil
	}
}

func goFlush(n *DNode) {
	if n.data != nil {
		Lock()
		p_out("%s", n.data[:min(10, len(n.data))])
		dataBlocks := make([]string, 0)
		for off := uint64(0); off < uint64(len(n.data)); {
			nextOff := off + getNextChunkOff(n.data[off:], uint64(len(n.data))-uint64(off))
			p_out("cur off: %d, next off: %d\n", off, nextOff)
			dataBlock := n.data[off:nextOff]
			key, _ := BytesSHA1(dataBlock)
			putData(key, dataBlock)
			dataBlocks = append(dataBlocks, key)
			off = nextOff
		}
		n.DataBlocks = dataBlocks
		n.dirty = false
		update(n)
		Unlock()
	}
}

func (n *DNode) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	// write data
	p_out("Data flushing...")
	if n.dirty {
		go goFlush(n)
		p_out("flush [%v] %q (dirty: %t, now %d bytes)\n", req, n.Name, n.dirty, len(n.data))
	}
	return nil
}

// hard links. Note that 'name' is not modified, so potential debugging problem.
func (p *DNode) Link(ctx context.Context, req *fuse.LinkRequest, oldNode fs.Node) (fs.Node, error) {
	p_out("enter Link\n")
	if readOnly {
		return nil, fuse.EPERM
	} else {
		Lock()
		curDNode := oldNode.(*DNode)
		// Hard link of dir is not allowed
		if curDNode.isDir() {
			Unlock()
			return nil, fuse.EPERM
		} else {
			curTime := time.Now()
			curDNode.Attrs.Atime = curTime
			curDNode.Attrs.Mtime = curTime
			p.kids[req.NewName] = curDNode
			update(p)
			Unlock()
			return curDNode, nil
		}
	}
}

func (n *DNode) Remove(ctx context.Context, req *fuse.RemoveRequest) error {

	if readOnly {
		return fuse.EPERM
	} else {
		p_out("enter Remove\n")
		if _, res := n.kids[req.Name]; !res {
			// no such file or directory
			return fuse.ENOENT
		} else {
			// rm dir without req.Dir == 1
			if n.kids[req.Name].isDir() && !req.Dir {
				return fuse.EPERM
			} else {
				Lock()
				if n.loadFromDB {
					n.loadAllData()
				}
				delete(n.kids, req.Name)
				update(n)
				Unlock()
				return nil
			}
		}
		return nil
	}
}

func (p *DNode) Symlink(ctx context.Context, req *fuse.SymlinkRequest) (fs.Node, error) {
	p_out("Sym Link Call   -- receiver id %d, new name: %q, target name: %q\n", p.Attrs.Inode, req.NewName, req.Target)
	if readOnly {
		return nil, fuse.EPERM
	} else {
		Lock()
		d := new(DNode)
		d.init(req.NewName, os.ModeSymlink|0777)
		p.kids[req.NewName] = d
		d.data = []byte(req.Target)
		d.dirty = true
		d.Attrs.Size = uint64(len(d.data))
		Unlock()
		return d, nil
	}
}

func (n *DNode) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	// TODO:
	return "", nil
}

func (n *DNode) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	p_out("enter Rename\n")

	if readOnly {
		return fuse.EPERM
	} else {
		if _, res := n.kids[req.OldName]; !res {
			// file not found
			return fuse.ENOENT
		} else {
			Lock()
			newDNode := newDir.(*DNode)
			newDNode.kids[req.NewName] = n.kids[req.OldName]
			newDNode.kids[req.NewName].Name = req.NewName
			delete(n.kids, req.OldName)
			update(newDNode.kids[req.NewName])
			update(n)
			Unlock()
			return nil
		}
	}

}

func update(n *DNode) error {

	for curNode := n; curNode != nil; curNode = curNode.parent {
		if curNode.loadFromDB {
			curNode.loadAllData()
		}
		curTime := time.Now()
		curNode.Attrs.Mtime = curTime
		p_out("Node, time: %s name: %s, nid: %d needs updating", curTime.String(), curNode.Name, curNode.Attrs.Inode)
		curNode.metaDirty = true
	}

	return nil
}

func putData(key string, dataBlock []byte) error {
	//p_out("Put data, sha1: %s", key[:min(len(key), 10)])
	return db.Put([]byte(key), dataBlock, nil)
}

func getData(key string) ([]byte, error) {
	return db.Get([]byte(key), nil)
}

func BytesSHA1(bytes []byte) (string, error) {
	h := sha1.New()
	tmp := make([]byte, len(bytes))
	copy(tmp, bytes)
	h.Write(tmp)
	sha1 := h.Sum(nil)
	sha1str := base64.StdEncoding.EncodeToString(sha1)
	return sha1str, nil
}

func getNextChunkOff(buf []byte, len uint64) uint64 {
	THE_PRIME := uint64(31)
	HASHLEN := uint64(32)
	b := THE_PRIME
	b_n := uint64(1)
	saved := make([]uint64, 256)
	hash := uint64(0)
	MINCHUNK := uint64(2048)
	TARGETCHUNK := uint64(4096)
	MAXCHUNK := uint64(8192)

	if b > 0 {
		for i := uint64(0); i < HASHLEN-1; i++ {
			b_n *= b
		}
		for i := uint64(0); i < uint64(256); i++ {
			saved[i] = i * b_n
		}
	}

	off := uint64(0)
	for ; (off < HASHLEN) && (off < len); off++ {
		hash = hash*b + uint64(buf[off])
	}

	for off < len {
		hash = (hash-saved[buf[off-HASHLEN]])*b + uint64(buf[off])
		off++
		if ((off >= MINCHUNK) && ((hash % TARGETCHUNK) == 1)) || (off >= MAXCHUNK) {
			return off
		}
	}
	return off
}

func getHead() (*DNode, uint64) {
	b, err := db.Get([]byte("head"), nil)
	if err != nil {
		return nil, 0
	}

	if h := importProtoHead(b); h != nil {
		//p_out("ROOT ID: %s", h.Root[:10])
		//p_out("Get head, root id: %s, nextInd: %d", h.Root[:10], h.NextInd)
		root := getNode([]byte(h.Root))
		//p_out("Head Root SHA1: %s", h.Root[:10])
		root.loadAllData()
		return root, h.NextInd
	} else {
		fmt.Printf("Head not found.")
		return nil, 0
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func putHead() error {
	if headBuf, err := exportProtoHead(); err == nil {
		err := db.Put([]byte("head"), headBuf, nil)
		rootSig, _ := root.SHA1()
		p_out("Put Head, root:%s %s, nextInd: %d", root.sig[:min(len(root.sig), 10)], rootSig[:min(len(root.sig), 10)], nextInd)
		return err
	} else {
		return nil
	}
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

func getNode(name []byte) *DNode {
	if name == nil {
		return nil
	}
	b, err := db.Get(name, nil)
	if err != nil {
		return nil
	}

	n := new(DNode)
	if err = importProtoNode(b, n); err == nil {
		n.sig = string(name)
		return n
	} else {
		return nil
	}
}

func putNode(n *DNode) error {
	key := []byte(n.sig)
	if value, err := exportProtoNode(n); err == nil {
		err := db.Put(key, value, nil)
		if err == nil {
			p_out("Put node, key: %s, nid: %d", n.sig[:10], n.Attrs.Inode)
		} else {
			p_out("Fail to Put node, key: %s, nid: %d", n.sig[:10], n.Attrs.Inode)
		}
		return err
	} else {
		return err
	}
}

func importProtoNode(bytes []byte, out *DNode) error {
	in := new(pb.Node)
	if err := proto.Unmarshal(bytes, in); err != nil {
		return err
	}

	out.Name = in.Name
	out.Version = int(in.Version)
	out.PrevSig = in.PrevSig
	out.loadFromDB = true

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

func importProtoHead(bytes []byte) *pb.Head {
	in := new(pb.Head)
	if err := proto.Unmarshal(bytes, in); err != nil {
		return nil
	}
	return in
}

func exportProtoHead() ([]byte, error) {
	in := new(pb.Head)
	in.Root = root.sig
	in.NextInd = nextInd
	//p_out("Put head, root id: %s, nextInd: %d", in.Root[:10], nextInd)
	bytes, err := proto.Marshal(in)
	return bytes, err
}

// Returns last version before "tm", or nil
func (n *DNode) getTimedVersion(tm time.Time) *DNode {
	if n.Attrs.Mtime.Before(tm) {
		return n
	}
	if n2 := getNode([]byte(n.PrevSig)); n2 != nil {
		p_out(n2.Attrs.Mtime.String())
		return n2.getTimedVersion(tm)
	}
	return nil
}

func initStore(newfs bool, dbPath string, ldbCache int, ro bool) {
	o := &opt.Options{
		ReadOnly: ro,
	}
	db, err = leveldb.OpenFile(dbPath, o)
	if err != nil {
		fmt.Printf("Problem opening DB dir %q\n", dbPath)
	}
}

func closeStore() {
	/*
		pbHead := new(pb.Head)
		pbHead.Root = root.sig
		pbHead.NextInd = nextInd

		if headBuf, err := exportProtoHead(); err != nil {
			fmt.Printf("Exporting Head fails")
		} else {
			db.Put([]byte("head"), headBuf, nil)
		}
	*/

	db.Close()
}

func traversalSave(n *DNode) {
	if n.metaDirty {
		if n.loadFromDB {
			n.loadAllData()
		}
		if n.kids != nil {
			for _, kid := range n.kids {
				traversalSave(kid)
			}
		}
		n.ChildSigs = make(map[string]string)
		for kidName, kid := range n.kids {
			n.ChildSigs[kidName] = kid.sig
		}
		n.PrevSig = n.sig
		p_out("N PREVSIG %s", n.PrevSig)
		n.Version += 1
		sig, _ := n.SHA1()
		p_out("renew sig: %s", sig)
		n.sig = sig
		putNode(n)
		n.metaDirty = false
		p_out("Put node, Inode: %d, Name: %s", n.Attrs.Inode, n.Name)
	}
}

func Flusher() {
	for true {
		time.Sleep(5000 * time.Millisecond)
		p_out("Metadata flushing...")
		Lock()
		traversalSave(root)
		putHead()
		Unlock()
	}
}

func Init(dbg bool, cmp bool, mountPoint string, newfs bool, dbPath string, tmStr string, ro bool, ldbCache int) {
	// Initialize a new diskv store, rooted at "my-data-dir", with a 1MB cache.

	debug = dbg
	compress = cmp
	readOnly = ro && tmStr == ""

	initStore(newfs, dbPath, ldbCache, readOnly)

	//replicaID := uint64(rand.Int63())

	if n, ni := getHead(); n != nil {
		root = n

		if tmStr != "" {
			tmParsed, err := parseLocalTime(tmStr)
			if err != nil {
				p_err("Error parsing time %q\n", tmStr)
			}
			if n := root.getTimedVersion(tmParsed); n != nil {
				root = n
				p_out("Found ARCHIVAL ROOT at %v\n", root.Attrs.Mtime)
				readOnly = true
			} else {
				p_err("Did not find root prior to %v\n", tmStr)
			}
		}
		nextInd = ni
	} else {
		p_out("GETHEAD fail\n")
		root = new(DNode)
		root.init("", os.ModeDir|0755)
		root.Name = "ROOT"
	}
	p_out("root inode %v", root.Attrs.Inode)

	p_out("compress: %t\n", compress)

	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		p_err("mount pt creation fail\n")
	}

	fuse.Unmount(mountPoint)
	c, err := fuse.Mount(mountPoint)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go func() {
		<-ch
		defer c.Close()
		defer db.Close()
		fuse.Unmount(mountPoint)
		os.Exit(1)
	}()

	if !readOnly {
		go Flusher()
	}

	err = fs.Serve(c, FS{})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
