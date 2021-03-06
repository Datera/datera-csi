// Code generated by protoc-gen-go. DO NOT EDIT.
// source: iscsi-rpc.proto

package iscsi_rpc

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
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

type SendArgsRequest struct {
	Args                 string   `protobuf:"bytes,1,opt,name=args,proto3" json:"args,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SendArgsRequest) Reset()         { *m = SendArgsRequest{} }
func (m *SendArgsRequest) String() string { return proto.CompactTextString(m) }
func (*SendArgsRequest) ProtoMessage()    {}
func (*SendArgsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a8bcb1a32be894c2, []int{0}
}

func (m *SendArgsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SendArgsRequest.Unmarshal(m, b)
}
func (m *SendArgsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SendArgsRequest.Marshal(b, m, deterministic)
}
func (m *SendArgsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SendArgsRequest.Merge(m, src)
}
func (m *SendArgsRequest) XXX_Size() int {
	return xxx_messageInfo_SendArgsRequest.Size(m)
}
func (m *SendArgsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SendArgsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SendArgsRequest proto.InternalMessageInfo

func (m *SendArgsRequest) GetArgs() string {
	if m != nil {
		return m.Args
	}
	return ""
}

type SendArgsReply struct {
	Result               string   `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SendArgsReply) Reset()         { *m = SendArgsReply{} }
func (m *SendArgsReply) String() string { return proto.CompactTextString(m) }
func (*SendArgsReply) ProtoMessage()    {}
func (*SendArgsReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_a8bcb1a32be894c2, []int{1}
}

func (m *SendArgsReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SendArgsReply.Unmarshal(m, b)
}
func (m *SendArgsReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SendArgsReply.Marshal(b, m, deterministic)
}
func (m *SendArgsReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SendArgsReply.Merge(m, src)
}
func (m *SendArgsReply) XXX_Size() int {
	return xxx_messageInfo_SendArgsReply.Size(m)
}
func (m *SendArgsReply) XXX_DiscardUnknown() {
	xxx_messageInfo_SendArgsReply.DiscardUnknown(m)
}

var xxx_messageInfo_SendArgsReply proto.InternalMessageInfo

func (m *SendArgsReply) GetResult() string {
	if m != nil {
		return m.Result
	}
	return ""
}

type GetInitiatorNameRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetInitiatorNameRequest) Reset()         { *m = GetInitiatorNameRequest{} }
func (m *GetInitiatorNameRequest) String() string { return proto.CompactTextString(m) }
func (*GetInitiatorNameRequest) ProtoMessage()    {}
func (*GetInitiatorNameRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a8bcb1a32be894c2, []int{2}
}

func (m *GetInitiatorNameRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetInitiatorNameRequest.Unmarshal(m, b)
}
func (m *GetInitiatorNameRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetInitiatorNameRequest.Marshal(b, m, deterministic)
}
func (m *GetInitiatorNameRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetInitiatorNameRequest.Merge(m, src)
}
func (m *GetInitiatorNameRequest) XXX_Size() int {
	return xxx_messageInfo_GetInitiatorNameRequest.Size(m)
}
func (m *GetInitiatorNameRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetInitiatorNameRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetInitiatorNameRequest proto.InternalMessageInfo

type GetInitiatorNameReply struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetInitiatorNameReply) Reset()         { *m = GetInitiatorNameReply{} }
func (m *GetInitiatorNameReply) String() string { return proto.CompactTextString(m) }
func (*GetInitiatorNameReply) ProtoMessage()    {}
func (*GetInitiatorNameReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_a8bcb1a32be894c2, []int{3}
}

func (m *GetInitiatorNameReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetInitiatorNameReply.Unmarshal(m, b)
}
func (m *GetInitiatorNameReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetInitiatorNameReply.Marshal(b, m, deterministic)
}
func (m *GetInitiatorNameReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetInitiatorNameReply.Merge(m, src)
}
func (m *GetInitiatorNameReply) XXX_Size() int {
	return xxx_messageInfo_GetInitiatorNameReply.Size(m)
}
func (m *GetInitiatorNameReply) XXX_DiscardUnknown() {
	xxx_messageInfo_GetInitiatorNameReply.DiscardUnknown(m)
}

var xxx_messageInfo_GetInitiatorNameReply proto.InternalMessageInfo

func (m *GetInitiatorNameReply) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func init() {
	proto.RegisterType((*SendArgsRequest)(nil), "iscsi_rpc.SendArgsRequest")
	proto.RegisterType((*SendArgsReply)(nil), "iscsi_rpc.SendArgsReply")
	proto.RegisterType((*GetInitiatorNameRequest)(nil), "iscsi_rpc.GetInitiatorNameRequest")
	proto.RegisterType((*GetInitiatorNameReply)(nil), "iscsi_rpc.GetInitiatorNameReply")
}

func init() { proto.RegisterFile("iscsi-rpc.proto", fileDescriptor_a8bcb1a32be894c2) }

var fileDescriptor_a8bcb1a32be894c2 = []byte{
	// 208 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xcf, 0x2c, 0x4e, 0x2e,
	0xce, 0xd4, 0x2d, 0x2a, 0x48, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x04, 0x0b, 0xc4,
	0x17, 0x15, 0x24, 0x2b, 0xa9, 0x72, 0xf1, 0x07, 0xa7, 0xe6, 0xa5, 0x38, 0x16, 0xa5, 0x17, 0x07,
	0xa5, 0x16, 0x96, 0xa6, 0x16, 0x97, 0x08, 0x09, 0x71, 0xb1, 0x24, 0x16, 0xa5, 0x17, 0x4b, 0x30,
	0x2a, 0x30, 0x6a, 0x70, 0x06, 0x81, 0xd9, 0x4a, 0xea, 0x5c, 0xbc, 0x08, 0x65, 0x05, 0x39, 0x95,
	0x42, 0x62, 0x5c, 0x6c, 0x45, 0xa9, 0xc5, 0xa5, 0x39, 0x25, 0x50, 0x65, 0x50, 0x9e, 0x92, 0x24,
	0x97, 0xb8, 0x7b, 0x6a, 0x89, 0x67, 0x5e, 0x66, 0x49, 0x66, 0x62, 0x49, 0x7e, 0x91, 0x5f, 0x62,
	0x6e, 0x2a, 0xd4, 0x5c, 0x25, 0x6d, 0x2e, 0x51, 0x4c, 0x29, 0x90, 0x59, 0x42, 0x5c, 0x2c, 0x79,
	0x89, 0xb9, 0xa9, 0x30, 0x0b, 0x41, 0x6c, 0xa3, 0x55, 0x8c, 0x5c, 0x1c, 0x9e, 0x20, 0x57, 0x26,
	0xa6, 0xe4, 0x0a, 0x39, 0x71, 0x71, 0xc0, 0x6c, 0x17, 0x92, 0xd2, 0x83, 0x3b, 0x5e, 0x0f, 0xcd,
	0xe5, 0x52, 0x12, 0x58, 0xe5, 0x0a, 0x72, 0x2a, 0x95, 0x18, 0x84, 0xa2, 0xb8, 0x04, 0xd0, 0x6d,
	0x17, 0x52, 0x42, 0x52, 0x8f, 0xc3, 0xd5, 0x52, 0x0a, 0x78, 0xd5, 0x80, 0xcd, 0x4e, 0x62, 0x03,
	0x07, 0xab, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0xac, 0xe3, 0xe3, 0x9a, 0x69, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// IscsiadmClient is the client API for Iscsiadm service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type IscsiadmClient interface {
	SendArgs(ctx context.Context, in *SendArgsRequest, opts ...grpc.CallOption) (*SendArgsReply, error)
	GetInitiatorName(ctx context.Context, in *GetInitiatorNameRequest, opts ...grpc.CallOption) (*GetInitiatorNameReply, error)
}

type iscsiadmClient struct {
	cc *grpc.ClientConn
}

func NewIscsiadmClient(cc *grpc.ClientConn) IscsiadmClient {
	return &iscsiadmClient{cc}
}

func (c *iscsiadmClient) SendArgs(ctx context.Context, in *SendArgsRequest, opts ...grpc.CallOption) (*SendArgsReply, error) {
	out := new(SendArgsReply)
	err := c.cc.Invoke(ctx, "/iscsi_rpc.Iscsiadm/SendArgs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *iscsiadmClient) GetInitiatorName(ctx context.Context, in *GetInitiatorNameRequest, opts ...grpc.CallOption) (*GetInitiatorNameReply, error) {
	out := new(GetInitiatorNameReply)
	err := c.cc.Invoke(ctx, "/iscsi_rpc.Iscsiadm/GetInitiatorName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IscsiadmServer is the server API for Iscsiadm service.
type IscsiadmServer interface {
	SendArgs(context.Context, *SendArgsRequest) (*SendArgsReply, error)
	GetInitiatorName(context.Context, *GetInitiatorNameRequest) (*GetInitiatorNameReply, error)
}

func RegisterIscsiadmServer(s *grpc.Server, srv IscsiadmServer) {
	s.RegisterService(&_Iscsiadm_serviceDesc, srv)
}

func _Iscsiadm_SendArgs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SendArgsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IscsiadmServer).SendArgs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/iscsi_rpc.Iscsiadm/SendArgs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IscsiadmServer).SendArgs(ctx, req.(*SendArgsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Iscsiadm_GetInitiatorName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInitiatorNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IscsiadmServer).GetInitiatorName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/iscsi_rpc.Iscsiadm/GetInitiatorName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IscsiadmServer).GetInitiatorName(ctx, req.(*GetInitiatorNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Iscsiadm_serviceDesc = grpc.ServiceDesc{
	ServiceName: "iscsi_rpc.Iscsiadm",
	HandlerType: (*IscsiadmServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SendArgs",
			Handler:    _Iscsiadm_SendArgs_Handler,
		},
		{
			MethodName: "GetInitiatorName",
			Handler:    _Iscsiadm_GetInitiatorName_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "iscsi-rpc.proto",
}
