// Code generated by protoc-gen-go.
// source: cmdctrlsvc.proto
// DO NOT EDIT!

/*
Package cmdctrlsvc is a generated protocol buffer package.

It is generated from these files:
	cmdctrlsvc.proto

It has these top-level messages:
	RingUpdateResult
	StatsMsg
	EmptyMsg
	StatusMsg
	Ring
	HealthCheckMsg
*/
package cmdctrlsvc

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

type RingUpdateResult struct {
	Newversion int64 `protobuf:"varint,1,opt,name=newversion" json:"newversion,omitempty"`
}

func (m *RingUpdateResult) Reset()                    { *m = RingUpdateResult{} }
func (m *RingUpdateResult) String() string            { return proto.CompactTextString(m) }
func (*RingUpdateResult) ProtoMessage()               {}
func (*RingUpdateResult) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type StatsMsg struct {
	Statsjson []byte `protobuf:"bytes,1,opt,name=statsjson,proto3" json:"statsjson,omitempty"`
}

func (m *StatsMsg) Reset()                    { *m = StatsMsg{} }
func (m *StatsMsg) String() string            { return proto.CompactTextString(m) }
func (*StatsMsg) ProtoMessage()               {}
func (*StatsMsg) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type EmptyMsg struct {
}

func (m *EmptyMsg) Reset()                    { *m = EmptyMsg{} }
func (m *EmptyMsg) String() string            { return proto.CompactTextString(m) }
func (*EmptyMsg) ProtoMessage()               {}
func (*EmptyMsg) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

type StatusMsg struct {
	Status bool   `protobuf:"varint,1,opt,name=status" json:"status,omitempty"`
	Msg    string `protobuf:"bytes,2,opt,name=msg" json:"msg,omitempty"`
}

func (m *StatusMsg) Reset()                    { *m = StatusMsg{} }
func (m *StatusMsg) String() string            { return proto.CompactTextString(m) }
func (*StatusMsg) ProtoMessage()               {}
func (*StatusMsg) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

type Ring struct {
	Version int64  `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	Ring    []byte `protobuf:"bytes,2,opt,name=ring,proto3" json:"ring,omitempty"`
}

func (m *Ring) Reset()                    { *m = Ring{} }
func (m *Ring) String() string            { return proto.CompactTextString(m) }
func (*Ring) ProtoMessage()               {}
func (*Ring) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

type HealthCheckMsg struct {
	Status bool   `protobuf:"varint,1,opt,name=status" json:"status,omitempty"`
	Msg    string `protobuf:"bytes,2,opt,name=msg" json:"msg,omitempty"`
	Ts     int64  `protobuf:"varint,3,opt,name=ts" json:"ts,omitempty"`
}

func (m *HealthCheckMsg) Reset()                    { *m = HealthCheckMsg{} }
func (m *HealthCheckMsg) String() string            { return proto.CompactTextString(m) }
func (*HealthCheckMsg) ProtoMessage()               {}
func (*HealthCheckMsg) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func init() {
	proto.RegisterType((*RingUpdateResult)(nil), "cmdctrlsvc.RingUpdateResult")
	proto.RegisterType((*StatsMsg)(nil), "cmdctrlsvc.StatsMsg")
	proto.RegisterType((*EmptyMsg)(nil), "cmdctrlsvc.EmptyMsg")
	proto.RegisterType((*StatusMsg)(nil), "cmdctrlsvc.StatusMsg")
	proto.RegisterType((*Ring)(nil), "cmdctrlsvc.Ring")
	proto.RegisterType((*HealthCheckMsg)(nil), "cmdctrlsvc.HealthCheckMsg")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion2

// Client API for CmdCtrl service

type CmdCtrlClient interface {
	RingUpdate(ctx context.Context, in *Ring, opts ...grpc.CallOption) (*RingUpdateResult, error)
	Reload(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error)
	Restart(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error)
	Start(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error)
	Stop(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error)
	Exit(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error)
	Stats(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatsMsg, error)
	HealthCheck(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*HealthCheckMsg, error)
}

type cmdCtrlClient struct {
	cc *grpc.ClientConn
}

func NewCmdCtrlClient(cc *grpc.ClientConn) CmdCtrlClient {
	return &cmdCtrlClient{cc}
}

func (c *cmdCtrlClient) RingUpdate(ctx context.Context, in *Ring, opts ...grpc.CallOption) (*RingUpdateResult, error) {
	out := new(RingUpdateResult)
	err := grpc.Invoke(ctx, "/cmdctrlsvc.CmdCtrl/RingUpdate", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cmdCtrlClient) Reload(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error) {
	out := new(StatusMsg)
	err := grpc.Invoke(ctx, "/cmdctrlsvc.CmdCtrl/Reload", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cmdCtrlClient) Restart(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error) {
	out := new(StatusMsg)
	err := grpc.Invoke(ctx, "/cmdctrlsvc.CmdCtrl/Restart", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cmdCtrlClient) Start(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error) {
	out := new(StatusMsg)
	err := grpc.Invoke(ctx, "/cmdctrlsvc.CmdCtrl/Start", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cmdCtrlClient) Stop(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error) {
	out := new(StatusMsg)
	err := grpc.Invoke(ctx, "/cmdctrlsvc.CmdCtrl/Stop", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cmdCtrlClient) Exit(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatusMsg, error) {
	out := new(StatusMsg)
	err := grpc.Invoke(ctx, "/cmdctrlsvc.CmdCtrl/Exit", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cmdCtrlClient) Stats(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*StatsMsg, error) {
	out := new(StatsMsg)
	err := grpc.Invoke(ctx, "/cmdctrlsvc.CmdCtrl/Stats", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cmdCtrlClient) HealthCheck(ctx context.Context, in *EmptyMsg, opts ...grpc.CallOption) (*HealthCheckMsg, error) {
	out := new(HealthCheckMsg)
	err := grpc.Invoke(ctx, "/cmdctrlsvc.CmdCtrl/HealthCheck", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for CmdCtrl service

type CmdCtrlServer interface {
	RingUpdate(context.Context, *Ring) (*RingUpdateResult, error)
	Reload(context.Context, *EmptyMsg) (*StatusMsg, error)
	Restart(context.Context, *EmptyMsg) (*StatusMsg, error)
	Start(context.Context, *EmptyMsg) (*StatusMsg, error)
	Stop(context.Context, *EmptyMsg) (*StatusMsg, error)
	Exit(context.Context, *EmptyMsg) (*StatusMsg, error)
	Stats(context.Context, *EmptyMsg) (*StatsMsg, error)
	HealthCheck(context.Context, *EmptyMsg) (*HealthCheckMsg, error)
}

func RegisterCmdCtrlServer(s *grpc.Server, srv CmdCtrlServer) {
	s.RegisterService(&_CmdCtrl_serviceDesc, srv)
}

func _CmdCtrl_RingUpdate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Ring)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdCtrlServer).RingUpdate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cmdctrlsvc.CmdCtrl/RingUpdate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdCtrlServer).RingUpdate(ctx, req.(*Ring))
	}
	return interceptor(ctx, in, info, handler)
}

func _CmdCtrl_Reload_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdCtrlServer).Reload(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cmdctrlsvc.CmdCtrl/Reload",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdCtrlServer).Reload(ctx, req.(*EmptyMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _CmdCtrl_Restart_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdCtrlServer).Restart(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cmdctrlsvc.CmdCtrl/Restart",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdCtrlServer).Restart(ctx, req.(*EmptyMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _CmdCtrl_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdCtrlServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cmdctrlsvc.CmdCtrl/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdCtrlServer).Start(ctx, req.(*EmptyMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _CmdCtrl_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdCtrlServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cmdctrlsvc.CmdCtrl/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdCtrlServer).Stop(ctx, req.(*EmptyMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _CmdCtrl_Exit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdCtrlServer).Exit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cmdctrlsvc.CmdCtrl/Exit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdCtrlServer).Exit(ctx, req.(*EmptyMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _CmdCtrl_Stats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdCtrlServer).Stats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cmdctrlsvc.CmdCtrl/Stats",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdCtrlServer).Stats(ctx, req.(*EmptyMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _CmdCtrl_HealthCheck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdCtrlServer).HealthCheck(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cmdctrlsvc.CmdCtrl/HealthCheck",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdCtrlServer).HealthCheck(ctx, req.(*EmptyMsg))
	}
	return interceptor(ctx, in, info, handler)
}

var _CmdCtrl_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cmdctrlsvc.CmdCtrl",
	HandlerType: (*CmdCtrlServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RingUpdate",
			Handler:    _CmdCtrl_RingUpdate_Handler,
		},
		{
			MethodName: "Reload",
			Handler:    _CmdCtrl_Reload_Handler,
		},
		{
			MethodName: "Restart",
			Handler:    _CmdCtrl_Restart_Handler,
		},
		{
			MethodName: "Start",
			Handler:    _CmdCtrl_Start_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _CmdCtrl_Stop_Handler,
		},
		{
			MethodName: "Exit",
			Handler:    _CmdCtrl_Exit_Handler,
		},
		{
			MethodName: "Stats",
			Handler:    _CmdCtrl_Stats_Handler,
		},
		{
			MethodName: "HealthCheck",
			Handler:    _CmdCtrl_HealthCheck_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

var fileDescriptor0 = []byte{
	// 327 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x9c, 0x93, 0xcd, 0x6a, 0xf2, 0x40,
	0x14, 0x86, 0xd5, 0xf8, 0xfb, 0x7e, 0x22, 0x61, 0xf0, 0x2b, 0x41, 0xa4, 0x94, 0x59, 0xb9, 0x72,
	0x61, 0x6b, 0xdb, 0x55, 0xa1, 0x88, 0x50, 0x0a, 0xdd, 0x8c, 0xf4, 0x02, 0xd2, 0x64, 0x50, 0xdb,
	0xc4, 0x84, 0xcc, 0xd1, 0xb6, 0xf7, 0xda, 0x8b, 0xe9, 0x64, 0xd0, 0x3a, 0x0a, 0x2e, 0xcc, 0xee,
	0x9c, 0x87, 0xf3, 0xe4, 0x0c, 0x6f, 0x66, 0xe0, 0x06, 0x71, 0x18, 0x50, 0x16, 0xa9, 0x4d, 0x30,
	0x4c, 0xb3, 0x84, 0x12, 0x86, 0x3d, 0xe1, 0x23, 0xb8, 0x62, 0xb9, 0x9a, 0xbf, 0xa6, 0xa1, 0x4f,
	0x52, 0x48, 0xb5, 0x8e, 0x88, 0x5d, 0x02, 0x2b, 0xf9, 0xb9, 0x91, 0x99, 0x5a, 0x26, 0x2b, 0xaf,
	0x7c, 0x55, 0x1e, 0x38, 0xc2, 0x22, 0x7c, 0x80, 0xe6, 0x8c, 0x7c, 0x52, 0x2f, 0x6a, 0xce, 0xfa,
	0x68, 0xa9, 0xbc, 0x7e, 0x57, 0xdb, 0xd1, 0xb6, 0xd8, 0x03, 0x0e, 0x34, 0xa7, 0x71, 0x4a, 0xdf,
	0x7a, 0x92, 0x8f, 0xd1, 0xca, 0xad, 0xb5, 0xd1, 0x2e, 0x50, 0x57, 0xa6, 0x31, 0x4e, 0x53, 0x6c,
	0x3b, 0xe6, 0xc2, 0x89, 0xd5, 0xdc, 0xab, 0x68, 0xd8, 0x12, 0x79, 0xc9, 0x6f, 0x50, 0xcd, 0x0f,
	0xc8, 0x3c, 0x34, 0x0e, 0x4f, 0xb4, 0x6b, 0x19, 0x43, 0x35, 0xd3, 0x13, 0x46, 0x6a, 0x0b, 0x53,
	0xf3, 0x67, 0x74, 0x9e, 0xa4, 0x1f, 0xd1, 0x62, 0xb2, 0x90, 0xc1, 0xc7, 0x59, 0x1b, 0x59, 0x07,
	0x15, 0x52, 0x9e, 0x63, 0x96, 0xe8, 0x6a, 0xf4, 0xe3, 0xa0, 0x31, 0x89, 0xc3, 0x89, 0x4e, 0x8c,
	0x3d, 0x00, 0xfb, 0xb8, 0x98, 0x3b, 0xb4, 0xb2, 0xcd, 0x79, 0xaf, 0x7f, 0x4c, 0xec, 0x60, 0x79,
	0x89, 0xdd, 0xa1, 0x2e, 0x64, 0x94, 0xf8, 0x21, 0xeb, 0xda, 0x93, 0xbb, 0x90, 0x7a, 0xff, 0x6d,
	0xfa, 0x17, 0x97, 0x16, 0xef, 0xd1, 0xd0, 0x1f, 0x21, 0x3f, 0xa3, 0x73, 0xcd, 0x5b, 0xd4, 0x66,
	0x45, 0xbc, 0x31, 0xaa, 0x33, 0x4a, 0xd2, 0x02, 0xda, 0xf4, 0x6b, 0x59, 0x60, 0x5b, 0xcd, 0xdc,
	0xa9, 0x13, 0x5e, 0xf7, 0xd8, 0xdb, 0x6a, 0x8f, 0xf8, 0x67, 0xfd, 0xe7, 0x13, 0x72, 0xcf, 0xa6,
	0x87, 0xd7, 0x82, 0x97, 0xde, 0xea, 0xe6, 0x51, 0x5c, 0xff, 0x06, 0x00, 0x00, 0xff, 0xff, 0x0f,
	0xe7, 0xcb, 0xf2, 0x28, 0x03, 0x00, 0x00,
}
