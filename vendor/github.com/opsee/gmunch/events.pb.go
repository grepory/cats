// Code generated by protoc-gen-go.
// source: events.proto
// DO NOT EDIT!

/*
Package gmunch is a generated protocol buffer package.

It is generated from these files:
	events.proto

It has these top-level messages:
	Event
	Response
*/
package gmunch

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
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Event struct {
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Data []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
}

func (m *Event) Reset()                    { *m = Event{} }
func (m *Event) String() string            { return proto.CompactTextString(m) }
func (*Event) ProtoMessage()               {}
func (*Event) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type Response struct {
	Ok bool `protobuf:"varint,1,opt,name=ok" json:"ok,omitempty"`
}

func (m *Response) Reset()                    { *m = Response{} }
func (m *Response) String() string            { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()               {}
func (*Response) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func init() {
	proto.RegisterType((*Event)(nil), "gmunch.Event")
	proto.RegisterType((*Response)(nil), "gmunch.Response")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion3

// Client API for Events service

type EventsClient interface {
	Publish(ctx context.Context, in *Event, opts ...grpc.CallOption) (*Response, error)
}

type eventsClient struct {
	cc *grpc.ClientConn
}

func NewEventsClient(cc *grpc.ClientConn) EventsClient {
	return &eventsClient{cc}
}

func (c *eventsClient) Publish(ctx context.Context, in *Event, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := grpc.Invoke(ctx, "/gmunch.Events/Publish", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Events service

type EventsServer interface {
	Publish(context.Context, *Event) (*Response, error)
}

func RegisterEventsServer(s *grpc.Server, srv EventsServer) {
	s.RegisterService(&_Events_serviceDesc, srv)
}

func _Events_Publish_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Event)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventsServer).Publish(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gmunch.Events/Publish",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventsServer).Publish(ctx, req.(*Event))
	}
	return interceptor(ctx, in, info, handler)
}

var _Events_serviceDesc = grpc.ServiceDesc{
	ServiceName: "gmunch.Events",
	HandlerType: (*EventsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Publish",
			Handler:    _Events_Publish_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: fileDescriptor0,
}

func init() { proto.RegisterFile("events.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 146 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0x49, 0x2d, 0x4b, 0xcd,
	0x2b, 0x29, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x4b, 0xcf, 0x2d, 0xcd, 0x4b, 0xce,
	0x50, 0xd2, 0xe7, 0x62, 0x75, 0x05, 0x89, 0x0b, 0x09, 0x71, 0xb1, 0xe4, 0x25, 0xe6, 0xa6, 0x4a,
	0x30, 0x2a, 0x30, 0x6a, 0x70, 0x06, 0x81, 0xd9, 0x20, 0xb1, 0x94, 0xc4, 0x92, 0x44, 0x09, 0x26,
	0xa0, 0x18, 0x4f, 0x10, 0x98, 0xad, 0x24, 0xc5, 0xc5, 0x11, 0x94, 0x5a, 0x5c, 0x90, 0x9f, 0x57,
	0x9c, 0x2a, 0xc4, 0xc7, 0xc5, 0x94, 0x9f, 0x0d, 0xd6, 0xc1, 0x11, 0x04, 0x64, 0x19, 0x99, 0x71,
	0xb1, 0x81, 0x0d, 0x2b, 0x16, 0xd2, 0xe1, 0x62, 0x0f, 0x28, 0x4d, 0xca, 0xc9, 0x2c, 0xce, 0x10,
	0xe2, 0xd5, 0x83, 0x58, 0xa5, 0x07, 0x96, 0x92, 0x12, 0x80, 0x71, 0x61, 0xa6, 0x28, 0x31, 0x24,
	0xb1, 0x81, 0xdd, 0x64, 0x0c, 0x08, 0x00, 0x00, 0xff, 0xff, 0x9f, 0xe2, 0xd6, 0x5b, 0xa3, 0x00,
	0x00, 0x00,
}