// Code generated by protoc-gen-go. DO NOT EDIT.
// source: dfs.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Time struct {
	Sec                  int64    `protobuf:"varint,1,opt,name=sec,proto3" json:"sec,omitempty"`
	Nsec                 int32    `protobuf:"varint,2,opt,name=nsec,proto3" json:"nsec,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Time) Reset()         { *m = Time{} }
func (m *Time) String() string { return proto.CompactTextString(m) }
func (*Time) ProtoMessage()    {}
func (*Time) Descriptor() ([]byte, []int) {
	return fileDescriptor_968f35ddaa593aaf, []int{0}
}

func (m *Time) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Time.Unmarshal(m, b)
}
func (m *Time) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Time.Marshal(b, m, deterministic)
}
func (m *Time) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Time.Merge(m, src)
}
func (m *Time) XXX_Size() int {
	return xxx_messageInfo_Time.Size(m)
}
func (m *Time) XXX_DiscardUnknown() {
	xxx_messageInfo_Time.DiscardUnknown(m)
}

var xxx_messageInfo_Time proto.InternalMessageInfo

func (m *Time) GetSec() int64 {
	if m != nil {
		return m.Sec
	}
	return 0
}

func (m *Time) GetNsec() int32 {
	if m != nil {
		return m.Nsec
	}
	return 0
}

type Attrtype struct {
	Valid                int64    `protobuf:"varint,1,opt,name=Valid,proto3" json:"Valid,omitempty"`
	Inode                uint64   `protobuf:"varint,2,opt,name=Inode,proto3" json:"Inode,omitempty"`
	Size                 uint64   `protobuf:"varint,3,opt,name=Size,proto3" json:"Size,omitempty"`
	Blocks               uint64   `protobuf:"varint,4,opt,name=Blocks,proto3" json:"Blocks,omitempty"`
	Atime                uint64   `protobuf:"varint,5,opt,name=Atime,proto3" json:"Atime,omitempty"`
	Mtime                uint64   `protobuf:"varint,6,opt,name=Mtime,proto3" json:"Mtime,omitempty"`
	Ctime                uint64   `protobuf:"varint,7,opt,name=Ctime,proto3" json:"Ctime,omitempty"`
	Crtime               uint64   `protobuf:"varint,8,opt,name=Crtime,proto3" json:"Crtime,omitempty"`
	FileMode             uint32   `protobuf:"varint,9,opt,name=FileMode,proto3" json:"FileMode,omitempty"`
	Nlink                uint32   `protobuf:"varint,10,opt,name=Nlink,proto3" json:"Nlink,omitempty"`
	Uid                  uint32   `protobuf:"varint,11,opt,name=Uid,proto3" json:"Uid,omitempty"`
	Gid                  uint32   `protobuf:"varint,12,opt,name=Gid,proto3" json:"Gid,omitempty"`
	Rdev                 uint32   `protobuf:"varint,13,opt,name=Rdev,proto3" json:"Rdev,omitempty"`
	Flags                uint32   `protobuf:"varint,14,opt,name=Flags,proto3" json:"Flags,omitempty"`
	BlockSize            uint32   `protobuf:"varint,15,opt,name=BlockSize,proto3" json:"BlockSize,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Attrtype) Reset()         { *m = Attrtype{} }
func (m *Attrtype) String() string { return proto.CompactTextString(m) }
func (*Attrtype) ProtoMessage()    {}
func (*Attrtype) Descriptor() ([]byte, []int) {
	return fileDescriptor_968f35ddaa593aaf, []int{1}
}

func (m *Attrtype) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Attrtype.Unmarshal(m, b)
}
func (m *Attrtype) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Attrtype.Marshal(b, m, deterministic)
}
func (m *Attrtype) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Attrtype.Merge(m, src)
}
func (m *Attrtype) XXX_Size() int {
	return xxx_messageInfo_Attrtype.Size(m)
}
func (m *Attrtype) XXX_DiscardUnknown() {
	xxx_messageInfo_Attrtype.DiscardUnknown(m)
}

var xxx_messageInfo_Attrtype proto.InternalMessageInfo

func (m *Attrtype) GetValid() int64 {
	if m != nil {
		return m.Valid
	}
	return 0
}

func (m *Attrtype) GetInode() uint64 {
	if m != nil {
		return m.Inode
	}
	return 0
}

func (m *Attrtype) GetSize() uint64 {
	if m != nil {
		return m.Size
	}
	return 0
}

func (m *Attrtype) GetBlocks() uint64 {
	if m != nil {
		return m.Blocks
	}
	return 0
}

func (m *Attrtype) GetAtime() uint64 {
	if m != nil {
		return m.Atime
	}
	return 0
}

func (m *Attrtype) GetMtime() uint64 {
	if m != nil {
		return m.Mtime
	}
	return 0
}

func (m *Attrtype) GetCtime() uint64 {
	if m != nil {
		return m.Ctime
	}
	return 0
}

func (m *Attrtype) GetCrtime() uint64 {
	if m != nil {
		return m.Crtime
	}
	return 0
}

func (m *Attrtype) GetFileMode() uint32 {
	if m != nil {
		return m.FileMode
	}
	return 0
}

func (m *Attrtype) GetNlink() uint32 {
	if m != nil {
		return m.Nlink
	}
	return 0
}

func (m *Attrtype) GetUid() uint32 {
	if m != nil {
		return m.Uid
	}
	return 0
}

func (m *Attrtype) GetGid() uint32 {
	if m != nil {
		return m.Gid
	}
	return 0
}

func (m *Attrtype) GetRdev() uint32 {
	if m != nil {
		return m.Rdev
	}
	return 0
}

func (m *Attrtype) GetFlags() uint32 {
	if m != nil {
		return m.Flags
	}
	return 0
}

func (m *Attrtype) GetBlockSize() uint32 {
	if m != nil {
		return m.BlockSize
	}
	return 0
}

type Node struct {
	Name                 string            `protobuf:"bytes,1,opt,name=Name,proto3" json:"Name,omitempty"`
	Attrs                *Attrtype         `protobuf:"bytes,2,opt,name=Attrs,proto3" json:"Attrs,omitempty"`
	Version              uint64            `protobuf:"varint,3,opt,name=Version,proto3" json:"Version,omitempty"`
	PrevSig              string            `protobuf:"bytes,4,opt,name=PrevSig,proto3" json:"PrevSig,omitempty"`
	ChildSigs            map[string]string `protobuf:"bytes,5,rep,name=ChildSigs,proto3" json:"ChildSigs,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	DataBlocks           []string          `protobuf:"bytes,6,rep,name=DataBlocks,proto3" json:"DataBlocks,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Node) Reset()         { *m = Node{} }
func (m *Node) String() string { return proto.CompactTextString(m) }
func (*Node) ProtoMessage()    {}
func (*Node) Descriptor() ([]byte, []int) {
	return fileDescriptor_968f35ddaa593aaf, []int{2}
}

func (m *Node) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Node.Unmarshal(m, b)
}
func (m *Node) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Node.Marshal(b, m, deterministic)
}
func (m *Node) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Node.Merge(m, src)
}
func (m *Node) XXX_Size() int {
	return xxx_messageInfo_Node.Size(m)
}
func (m *Node) XXX_DiscardUnknown() {
	xxx_messageInfo_Node.DiscardUnknown(m)
}

var xxx_messageInfo_Node proto.InternalMessageInfo

func (m *Node) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Node) GetAttrs() *Attrtype {
	if m != nil {
		return m.Attrs
	}
	return nil
}

func (m *Node) GetVersion() uint64 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *Node) GetPrevSig() string {
	if m != nil {
		return m.PrevSig
	}
	return ""
}

func (m *Node) GetChildSigs() map[string]string {
	if m != nil {
		return m.ChildSigs
	}
	return nil
}

func (m *Node) GetDataBlocks() []string {
	if m != nil {
		return m.DataBlocks
	}
	return nil
}

type Head struct {
	Root                 string   `protobuf:"bytes,1,opt,name=Root,proto3" json:"Root,omitempty"`
	NextInd              uint64   `protobuf:"varint,2,opt,name=NextInd,proto3" json:"NextInd,omitempty"`
	Replica              uint64   `protobuf:"varint,3,opt,name=Replica,proto3" json:"Replica,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Head) Reset()         { *m = Head{} }
func (m *Head) String() string { return proto.CompactTextString(m) }
func (*Head) ProtoMessage()    {}
func (*Head) Descriptor() ([]byte, []int) {
	return fileDescriptor_968f35ddaa593aaf, []int{3}
}

func (m *Head) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Head.Unmarshal(m, b)
}
func (m *Head) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Head.Marshal(b, m, deterministic)
}
func (m *Head) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Head.Merge(m, src)
}
func (m *Head) XXX_Size() int {
	return xxx_messageInfo_Head.Size(m)
}
func (m *Head) XXX_DiscardUnknown() {
	xxx_messageInfo_Head.DiscardUnknown(m)
}

var xxx_messageInfo_Head proto.InternalMessageInfo

func (m *Head) GetRoot() string {
	if m != nil {
		return m.Root
	}
	return ""
}

func (m *Head) GetNextInd() uint64 {
	if m != nil {
		return m.NextInd
	}
	return 0
}

func (m *Head) GetReplica() uint64 {
	if m != nil {
		return m.Replica
	}
	return 0
}

func init() {
	proto.RegisterType((*Time)(nil), "pb.Time")
	proto.RegisterType((*Attrtype)(nil), "pb.Attrtype")
	proto.RegisterType((*Node)(nil), "pb.Node")
	proto.RegisterMapType((map[string]string)(nil), "pb.Node.ChildSigsEntry")
	proto.RegisterType((*Head)(nil), "pb.Head")
}

func init() { proto.RegisterFile("dfs.proto", fileDescriptor_968f35ddaa593aaf) }

var fileDescriptor_968f35ddaa593aaf = []byte{
	// 438 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x92, 0x5f, 0x6f, 0xd3, 0x30,
	0x14, 0xc5, 0x95, 0x3f, 0xed, 0x9a, 0xdb, 0x6d, 0x20, 0x0b, 0x81, 0x35, 0x21, 0x14, 0xe5, 0x29,
	0x0f, 0x28, 0x0f, 0x43, 0x48, 0x08, 0xf1, 0x32, 0x0a, 0x1b, 0x7b, 0x68, 0x84, 0x5c, 0xd8, 0xbb,
	0x5b, 0x9b, 0x62, 0xd5, 0x4d, 0xa2, 0xc4, 0x54, 0x94, 0x2f, 0xc0, 0x77, 0xe4, 0xd3, 0xa0, 0x7b,
	0xed, 0x6c, 0xf0, 0x76, 0xce, 0xef, 0xc6, 0x47, 0xb9, 0xc7, 0x86, 0x4c, 0x7d, 0x1b, 0xaa, 0xae,
	0x6f, 0x5d, 0xcb, 0xe2, 0x6e, 0x5d, 0xbc, 0x84, 0xf4, 0x8b, 0xd9, 0x6b, 0xf6, 0x18, 0x92, 0x41,
	0x6f, 0x78, 0x94, 0x47, 0x65, 0x22, 0x50, 0x32, 0x06, 0x69, 0x83, 0x28, 0xce, 0xa3, 0x72, 0x22,
	0x48, 0x17, 0x7f, 0x62, 0x98, 0x5d, 0x39, 0xd7, 0xbb, 0x63, 0xa7, 0xd9, 0x13, 0x98, 0xdc, 0x49,
	0x6b, 0x54, 0x38, 0xe4, 0x0d, 0xd2, 0xdb, 0xa6, 0x55, 0x9a, 0xce, 0xa5, 0xc2, 0x1b, 0x0c, 0x5b,
	0x99, 0x5f, 0x9a, 0x27, 0x04, 0x49, 0xb3, 0xa7, 0x30, 0x7d, 0x6f, 0xdb, 0xcd, 0x6e, 0xe0, 0x29,
	0xd1, 0xe0, 0x30, 0xe1, 0xca, 0x99, 0xbd, 0xe6, 0x13, 0x9f, 0x40, 0x06, 0xe9, 0x92, 0xe8, 0xd4,
	0xd3, 0xe5, 0x48, 0x17, 0x44, 0x4f, 0x3c, 0x25, 0x83, 0xc9, 0x8b, 0x9e, 0xf0, 0xcc, 0x27, 0x7b,
	0xc7, 0x2e, 0x60, 0x76, 0x6d, 0xac, 0x5e, 0xe2, 0xef, 0x65, 0x79, 0x54, 0x9e, 0x89, 0x7b, 0x8f,
	0x49, 0xb5, 0x35, 0xcd, 0x8e, 0x03, 0x0d, 0xbc, 0xc1, 0x5a, 0xbe, 0x1a, 0xc5, 0xe7, 0xc4, 0x50,
	0x22, 0xb9, 0x31, 0x8a, 0x9f, 0x7a, 0x72, 0x63, 0x14, 0xee, 0x26, 0x94, 0x3e, 0xf0, 0x33, 0x42,
	0xa4, 0x31, 0xed, 0xda, 0xca, 0xed, 0xc0, 0xcf, 0x7d, 0x1a, 0x19, 0xf6, 0x1c, 0x32, 0xda, 0x91,
	0xaa, 0x78, 0x44, 0x93, 0x07, 0x50, 0xfc, 0x8e, 0x21, 0xad, 0x43, 0x59, 0xb5, 0xdc, 0x6b, 0xea,
	0x35, 0x13, 0xa4, 0x59, 0x81, 0xa5, 0xb8, 0x7e, 0xa0, 0x5a, 0xe7, 0x97, 0xa7, 0x55, 0xb7, 0xae,
	0xc6, 0x9b, 0x10, 0x7e, 0xc4, 0x38, 0x9c, 0xdc, 0xe9, 0x7e, 0x30, 0x6d, 0x13, 0x7a, 0x1e, 0x2d,
	0x4e, 0x3e, 0xf7, 0xfa, 0xb0, 0x32, 0x5b, 0xea, 0x3a, 0x13, 0xa3, 0x65, 0xaf, 0x21, 0x5b, 0x7c,
	0x37, 0x56, 0xad, 0xcc, 0x76, 0xe0, 0x93, 0x3c, 0x29, 0xe7, 0x97, 0xcf, 0x30, 0x1b, 0x7f, 0xa4,
	0xba, 0x9f, 0x7c, 0x6c, 0x5c, 0x7f, 0x14, 0x0f, 0x5f, 0xb2, 0x17, 0x00, 0x1f, 0xa4, 0x93, 0xe1,
	0xfe, 0xa6, 0x79, 0x52, 0x66, 0xe2, 0x1f, 0x72, 0xf1, 0x0e, 0xce, 0xff, 0x3f, 0x8c, 0xbd, 0xed,
	0xf4, 0x31, 0xec, 0x84, 0x12, 0x3b, 0x3a, 0x48, 0xfb, 0xc3, 0xbf, 0x94, 0x4c, 0x78, 0xf3, 0x36,
	0x7e, 0x13, 0x15, 0x35, 0xa4, 0x9f, 0xb4, 0xf4, 0xcd, 0xb6, 0xad, 0x1b, 0x8b, 0x40, 0x8d, 0xab,
	0xd4, 0xfa, 0xa7, 0xbb, 0x6d, 0x54, 0x78, 0x61, 0xa3, 0xc5, 0x89, 0xd0, 0x9d, 0x35, 0x1b, 0x39,
	0xae, 0x1f, 0xec, 0x7a, 0x4a, 0xef, 0xfd, 0xd5, 0xdf, 0x00, 0x00, 0x00, 0xff, 0xff, 0x45, 0x49,
	0xe5, 0x06, 0xfc, 0x02, 0x00, 0x00,
}