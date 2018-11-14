// Code generated by protoc-gen-go. DO NOT EDIT.
// source: pb/hare.proto

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

type InnerMessage_Type int32

const (
	InnerMessage_STATUS   InnerMessage_Type = 0
	InnerMessage_PROPOSAL InnerMessage_Type = 1
	InnerMessage_COMMIT   InnerMessage_Type = 2
	InnerMessage_NOTIFY   InnerMessage_Type = 3
)

var InnerMessage_Type_name = map[int32]string{
	0: "STATUS",
	1: "PROPOSAL",
	2: "COMMIT",
	3: "NOTIFY",
}

var InnerMessage_Type_value = map[string]int32{
	"STATUS":   0,
	"PROPOSAL": 1,
	"COMMIT":   2,
	"NOTIFY":   3,
}

func (x InnerMessage_Type) String() string {
	return proto.EnumName(InnerMessage_Type_name, int32(x))
}

func (InnerMessage_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_03c7555ad582658b, []int{3, 0}
}

// top message of the protocol
type HareMessage struct {
	PubKey               []byte        `protobuf:"bytes,1,opt,name=pubKey,proto3" json:"pubKey,omitempty"`
	InnerSig             []byte        `protobuf:"bytes,2,opt,name=innerSig,proto3" json:"innerSig,omitempty"`
	Message              *InnerMessage `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	OuterSig             []byte        `protobuf:"bytes,4,opt,name=outerSig,proto3" json:"outerSig,omitempty"`
	Cert                 *Certificate  `protobuf:"bytes,5,opt,name=cert,proto3" json:"cert,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *HareMessage) Reset()         { *m = HareMessage{} }
func (m *HareMessage) String() string { return proto.CompactTextString(m) }
func (*HareMessage) ProtoMessage()    {}
func (*HareMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_03c7555ad582658b, []int{0}
}

func (m *HareMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_HareMessage.Unmarshal(m, b)
}
func (m *HareMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_HareMessage.Marshal(b, m, deterministic)
}
func (m *HareMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_HareMessage.Merge(m, src)
}
func (m *HareMessage) XXX_Size() int {
	return xxx_messageInfo_HareMessage.Size(m)
}
func (m *HareMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_HareMessage.DiscardUnknown(m)
}

var xxx_messageInfo_HareMessage proto.InternalMessageInfo

func (m *HareMessage) GetPubKey() []byte {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *HareMessage) GetInnerSig() []byte {
	if m != nil {
		return m.InnerSig
	}
	return nil
}

func (m *HareMessage) GetMessage() *InnerMessage {
	if m != nil {
		return m.Message
	}
	return nil
}

func (m *HareMessage) GetOuterSig() []byte {
	if m != nil {
		return m.OuterSig
	}
	return nil
}

func (m *HareMessage) GetCert() *Certificate {
	if m != nil {
		return m.Cert
	}
	return nil
}

// the certificate
type Certificate struct {
	Blocks               [][]byte       `protobuf:"bytes,1,rep,name=blocks,proto3" json:"blocks,omitempty"`
	Commits              []*HareMessage `protobuf:"bytes,2,rep,name=commits,proto3" json:"commits,omitempty"`
	AggSig               []byte         `protobuf:"bytes,3,opt,name=aggSig,proto3" json:"aggSig,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *Certificate) Reset()         { *m = Certificate{} }
func (m *Certificate) String() string { return proto.CompactTextString(m) }
func (*Certificate) ProtoMessage()    {}
func (*Certificate) Descriptor() ([]byte, []int) {
	return fileDescriptor_03c7555ad582658b, []int{1}
}

func (m *Certificate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Certificate.Unmarshal(m, b)
}
func (m *Certificate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Certificate.Marshal(b, m, deterministic)
}
func (m *Certificate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Certificate.Merge(m, src)
}
func (m *Certificate) XXX_Size() int {
	return xxx_messageInfo_Certificate.Size(m)
}
func (m *Certificate) XXX_DiscardUnknown() {
	xxx_messageInfo_Certificate.DiscardUnknown(m)
}

var xxx_messageInfo_Certificate proto.InternalMessageInfo

func (m *Certificate) GetBlocks() [][]byte {
	if m != nil {
		return m.Blocks
	}
	return nil
}

func (m *Certificate) GetCommits() []*HareMessage {
	if m != nil {
		return m.Commits
	}
	return nil
}

func (m *Certificate) GetAggSig() []byte {
	if m != nil {
		return m.AggSig
	}
	return nil
}

// safe value proof message
type SVP struct {
	Statuses             []*HareMessage `protobuf:"bytes,1,rep,name=statuses,proto3" json:"statuses,omitempty"`
	AggSig               []byte         `protobuf:"bytes,2,opt,name=aggSig,proto3" json:"aggSig,omitempty"`
	Cert                 *Certificate   `protobuf:"bytes,3,opt,name=cert,proto3" json:"cert,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *SVP) Reset()         { *m = SVP{} }
func (m *SVP) String() string { return proto.CompactTextString(m) }
func (*SVP) ProtoMessage()    {}
func (*SVP) Descriptor() ([]byte, []int) {
	return fileDescriptor_03c7555ad582658b, []int{2}
}

func (m *SVP) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SVP.Unmarshal(m, b)
}
func (m *SVP) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SVP.Marshal(b, m, deterministic)
}
func (m *SVP) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SVP.Merge(m, src)
}
func (m *SVP) XXX_Size() int {
	return xxx_messageInfo_SVP.Size(m)
}
func (m *SVP) XXX_DiscardUnknown() {
	xxx_messageInfo_SVP.DiscardUnknown(m)
}

var xxx_messageInfo_SVP proto.InternalMessageInfo

func (m *SVP) GetStatuses() []*HareMessage {
	if m != nil {
		return m.Statuses
	}
	return nil
}

func (m *SVP) GetAggSig() []byte {
	if m != nil {
		return m.AggSig
	}
	return nil
}

func (m *SVP) GetCert() *Certificate {
	if m != nil {
		return m.Cert
	}
	return nil
}

// basic message
type InnerMessage struct {
	Type                 InnerMessage_Type `protobuf:"varint,1,opt,name=type,proto3,enum=pb.InnerMessage_Type" json:"type,omitempty"`
	Layer                []byte            `protobuf:"bytes,2,opt,name=layer,proto3" json:"layer,omitempty"`
	K                    uint32            `protobuf:"varint,3,opt,name=k,proto3" json:"k,omitempty"`
	Ki                   uint32            `protobuf:"varint,4,opt,name=ki,proto3" json:"ki,omitempty"`
	Blocks               [][]byte          `protobuf:"bytes,5,rep,name=blocks,proto3" json:"blocks,omitempty"`
	RoleProof            []byte            `protobuf:"bytes,6,opt,name=roleProof,proto3" json:"roleProof,omitempty"`
	SvpProof             *SVP              `protobuf:"bytes,7,opt,name=svpProof,proto3" json:"svpProof,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *InnerMessage) Reset()         { *m = InnerMessage{} }
func (m *InnerMessage) String() string { return proto.CompactTextString(m) }
func (*InnerMessage) ProtoMessage()    {}
func (*InnerMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_03c7555ad582658b, []int{3}
}

func (m *InnerMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_InnerMessage.Unmarshal(m, b)
}
func (m *InnerMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_InnerMessage.Marshal(b, m, deterministic)
}
func (m *InnerMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InnerMessage.Merge(m, src)
}
func (m *InnerMessage) XXX_Size() int {
	return xxx_messageInfo_InnerMessage.Size(m)
}
func (m *InnerMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_InnerMessage.DiscardUnknown(m)
}

var xxx_messageInfo_InnerMessage proto.InternalMessageInfo

func (m *InnerMessage) GetType() InnerMessage_Type {
	if m != nil {
		return m.Type
	}
	return InnerMessage_STATUS
}

func (m *InnerMessage) GetLayer() []byte {
	if m != nil {
		return m.Layer
	}
	return nil
}

func (m *InnerMessage) GetK() uint32 {
	if m != nil {
		return m.K
	}
	return 0
}

func (m *InnerMessage) GetKi() uint32 {
	if m != nil {
		return m.Ki
	}
	return 0
}

func (m *InnerMessage) GetBlocks() [][]byte {
	if m != nil {
		return m.Blocks
	}
	return nil
}

func (m *InnerMessage) GetRoleProof() []byte {
	if m != nil {
		return m.RoleProof
	}
	return nil
}

func (m *InnerMessage) GetSvpProof() *SVP {
	if m != nil {
		return m.SvpProof
	}
	return nil
}

func init() {
	proto.RegisterEnum("pb.InnerMessage_Type", InnerMessage_Type_name, InnerMessage_Type_value)
	proto.RegisterType((*HareMessage)(nil), "pb.HareMessage")
	proto.RegisterType((*Certificate)(nil), "pb.Certificate")
	proto.RegisterType((*SVP)(nil), "pb.SVP")
	proto.RegisterType((*InnerMessage)(nil), "pb.InnerMessage")
}

func init() { proto.RegisterFile("pb/hare.proto", fileDescriptor_03c7555ad582658b) }

var fileDescriptor_03c7555ad582658b = []byte{
	// 406 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x92, 0x4f, 0x8f, 0x94, 0x30,
	0x18, 0xc6, 0x6d, 0x61, 0xfe, 0xf8, 0x0e, 0xb3, 0x92, 0x46, 0x0d, 0x31, 0x1e, 0x08, 0x7b, 0xd9,
	0xd5, 0x04, 0x93, 0xf5, 0xe2, 0x75, 0xdd, 0xc4, 0x38, 0xd1, 0x11, 0x52, 0x70, 0x13, 0xbd, 0x01,
	0xe9, 0xb2, 0x84, 0x99, 0x69, 0x53, 0x3a, 0x26, 0x5c, 0xfd, 0x40, 0x7e, 0x46, 0xd3, 0x76, 0x40,
	0x34, 0x66, 0x6f, 0x3c, 0x7d, 0xca, 0xef, 0x7d, 0x9f, 0x07, 0x60, 0x2d, 0xca, 0x37, 0xf7, 0x85,
	0x64, 0xb1, 0x90, 0x5c, 0x71, 0x82, 0x45, 0x19, 0xfd, 0x42, 0xb0, 0xfa, 0x58, 0x48, 0xb6, 0x65,
	0x5d, 0x57, 0xd4, 0x8c, 0x3c, 0x87, 0xb9, 0x38, 0x96, 0x9f, 0x58, 0x1f, 0xa0, 0x10, 0x5d, 0x78,
	0xf4, 0xa4, 0xc8, 0x0b, 0x58, 0x36, 0x87, 0x03, 0x93, 0x59, 0x53, 0x07, 0xd8, 0x38, 0xa3, 0x26,
	0xaf, 0x60, 0xb1, 0xb7, 0xaf, 0x07, 0x4e, 0x88, 0x2e, 0x56, 0x57, 0x7e, 0x2c, 0xca, 0x78, 0xa3,
	0xed, 0x13, 0x96, 0x0e, 0x17, 0x34, 0x87, 0x1f, 0x95, 0xe5, 0xb8, 0x96, 0x33, 0x68, 0x72, 0x0e,
	0x6e, 0xc5, 0xa4, 0x0a, 0x66, 0x06, 0xf2, 0x44, 0x43, 0x6e, 0x98, 0x54, 0xcd, 0x5d, 0x53, 0x15,
	0x8a, 0x51, 0x63, 0x46, 0xf7, 0xb0, 0x9a, 0x1c, 0xea, 0x7d, 0xcb, 0x1d, 0xaf, 0xda, 0x2e, 0x40,
	0xa1, 0xa3, 0xf7, 0xb5, 0x8a, 0x5c, 0xc2, 0xa2, 0xe2, 0xfb, 0x7d, 0xa3, 0xba, 0x00, 0x87, 0xce,
	0x80, 0x9b, 0x24, 0xa5, 0x83, 0xaf, 0x11, 0x45, 0x5d, 0xeb, 0x85, 0x1c, 0x1b, 0xd9, 0xaa, 0x88,
	0x83, 0x93, 0xdd, 0xa6, 0xe4, 0x35, 0x2c, 0x3b, 0x55, 0xa8, 0x63, 0xc7, 0xec, 0x8c, 0xff, 0xa0,
	0xc6, 0x0b, 0x13, 0x16, 0x9e, 0xb2, 0xc6, 0x68, 0xce, 0x43, 0xd1, 0x7e, 0x62, 0xf0, 0xa6, 0xad,
	0x91, 0x4b, 0x70, 0x55, 0x2f, 0x98, 0xf9, 0x14, 0x67, 0x57, 0xcf, 0xfe, 0x6d, 0x35, 0xce, 0x7b,
	0xc1, 0xa8, 0xb9, 0x42, 0x9e, 0xc2, 0x6c, 0x57, 0xf4, 0x4c, 0x9e, 0xe6, 0x5a, 0x41, 0x3c, 0x40,
	0xad, 0x99, 0xb9, 0xa6, 0xa8, 0x25, 0x67, 0x80, 0xdb, 0xc6, 0xb4, 0xbe, 0xa6, 0xb8, 0x6d, 0x26,
	0xdd, 0xcd, 0xfe, 0xea, 0xee, 0x25, 0x3c, 0x96, 0x7c, 0xc7, 0x52, 0xc9, 0xf9, 0x5d, 0x30, 0x37,
	0xbc, 0x3f, 0x07, 0xe4, 0x1c, 0x96, 0xdd, 0x0f, 0x61, 0xcd, 0x85, 0x89, 0xb3, 0xd0, 0x8b, 0x65,
	0xb7, 0x29, 0x1d, 0x8d, 0xe8, 0x1d, 0xb8, 0x7a, 0x39, 0x02, 0x30, 0xcf, 0xf2, 0xeb, 0xfc, 0x6b,
	0xe6, 0x3f, 0x22, 0x1e, 0x2c, 0x53, 0x9a, 0xa4, 0x49, 0x76, 0xfd, 0xd9, 0x47, 0xda, 0xb9, 0x49,
	0xb6, 0xdb, 0x4d, 0xee, 0x63, 0xfd, 0xfc, 0x25, 0xc9, 0x37, 0x1f, 0xbe, 0xf9, 0xce, 0x7b, 0xf7,
	0x3b, 0x16, 0x65, 0x39, 0x37, 0x7f, 0xe8, 0xdb, 0xdf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x05, 0xc6,
	0x27, 0xd9, 0xb2, 0x02, 0x00, 0x00,
}
