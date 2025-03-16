// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: proto/gw-bank.proto

package gw_bank

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	BankService_RegisterClient_FullMethodName      = "/gw_bank.BankService/RegisterClient"
	BankService_UpdateClientDetails_FullMethodName = "/gw_bank.BankService/UpdateClientDetails"
	BankService_ViewBalance_FullMethodName         = "/gw_bank.BankService/ViewBalance"
	BankService_LockIdempotency_FullMethodName     = "/gw_bank.BankService/LockIdempotency"
	BankService_InitiateTransaction_FullMethodName = "/gw_bank.BankService/InitiateTransaction"
)

// BankServiceClient is the client API for BankService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BankServiceClient interface {
	// Register a new Client
	RegisterClient(ctx context.Context, in *ClientDetails, opts ...grpc.CallOption) (*RegisterResponse, error)
	// Update Client Details
	UpdateClientDetails(ctx context.Context, in *UpdateDetails, opts ...grpc.CallOption) (*UpdateResponse, error)
	// Func to view bank account balance
	ViewBalance(ctx context.Context, in *ViewBalanceRequest, opts ...grpc.CallOption) (*ViewBalanceResponse, error)
	// Lock for Idempotency
	LockIdempotency(ctx context.Context, in *LockIdempotencyRequest, opts ...grpc.CallOption) (*LockIdempotencyResponse, error)
	// Initiate transaction
	InitiateTransaction(ctx context.Context, in *InitiateTransactionRequest, opts ...grpc.CallOption) (*InitiateTransactionResponse, error)
}

type bankServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBankServiceClient(cc grpc.ClientConnInterface) BankServiceClient {
	return &bankServiceClient{cc}
}

func (c *bankServiceClient) RegisterClient(ctx context.Context, in *ClientDetails, opts ...grpc.CallOption) (*RegisterResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RegisterResponse)
	err := c.cc.Invoke(ctx, BankService_RegisterClient_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) UpdateClientDetails(ctx context.Context, in *UpdateDetails, opts ...grpc.CallOption) (*UpdateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateResponse)
	err := c.cc.Invoke(ctx, BankService_UpdateClientDetails_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) ViewBalance(ctx context.Context, in *ViewBalanceRequest, opts ...grpc.CallOption) (*ViewBalanceResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ViewBalanceResponse)
	err := c.cc.Invoke(ctx, BankService_ViewBalance_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) LockIdempotency(ctx context.Context, in *LockIdempotencyRequest, opts ...grpc.CallOption) (*LockIdempotencyResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(LockIdempotencyResponse)
	err := c.cc.Invoke(ctx, BankService_LockIdempotency_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) InitiateTransaction(ctx context.Context, in *InitiateTransactionRequest, opts ...grpc.CallOption) (*InitiateTransactionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InitiateTransactionResponse)
	err := c.cc.Invoke(ctx, BankService_InitiateTransaction_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BankServiceServer is the server API for BankService service.
// All implementations must embed UnimplementedBankServiceServer
// for forward compatibility.
type BankServiceServer interface {
	// Register a new Client
	RegisterClient(context.Context, *ClientDetails) (*RegisterResponse, error)
	// Update Client Details
	UpdateClientDetails(context.Context, *UpdateDetails) (*UpdateResponse, error)
	// Func to view bank account balance
	ViewBalance(context.Context, *ViewBalanceRequest) (*ViewBalanceResponse, error)
	// Lock for Idempotency
	LockIdempotency(context.Context, *LockIdempotencyRequest) (*LockIdempotencyResponse, error)
	// Initiate transaction
	InitiateTransaction(context.Context, *InitiateTransactionRequest) (*InitiateTransactionResponse, error)
	mustEmbedUnimplementedBankServiceServer()
}

// UnimplementedBankServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedBankServiceServer struct{}

func (UnimplementedBankServiceServer) RegisterClient(context.Context, *ClientDetails) (*RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterClient not implemented")
}
func (UnimplementedBankServiceServer) UpdateClientDetails(context.Context, *UpdateDetails) (*UpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateClientDetails not implemented")
}
func (UnimplementedBankServiceServer) ViewBalance(context.Context, *ViewBalanceRequest) (*ViewBalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ViewBalance not implemented")
}
func (UnimplementedBankServiceServer) LockIdempotency(context.Context, *LockIdempotencyRequest) (*LockIdempotencyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LockIdempotency not implemented")
}
func (UnimplementedBankServiceServer) InitiateTransaction(context.Context, *InitiateTransactionRequest) (*InitiateTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitiateTransaction not implemented")
}
func (UnimplementedBankServiceServer) mustEmbedUnimplementedBankServiceServer() {}
func (UnimplementedBankServiceServer) testEmbeddedByValue()                     {}

// UnsafeBankServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BankServiceServer will
// result in compilation errors.
type UnsafeBankServiceServer interface {
	mustEmbedUnimplementedBankServiceServer()
}

func RegisterBankServiceServer(s grpc.ServiceRegistrar, srv BankServiceServer) {
	// If the following call pancis, it indicates UnimplementedBankServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&BankService_ServiceDesc, srv)
}

func _BankService_RegisterClient_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClientDetails)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).RegisterClient(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_RegisterClient_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).RegisterClient(ctx, req.(*ClientDetails))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_UpdateClientDetails_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateDetails)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).UpdateClientDetails(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_UpdateClientDetails_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).UpdateClientDetails(ctx, req.(*UpdateDetails))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_ViewBalance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ViewBalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).ViewBalance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_ViewBalance_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).ViewBalance(ctx, req.(*ViewBalanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_LockIdempotency_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LockIdempotencyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).LockIdempotency(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_LockIdempotency_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).LockIdempotency(ctx, req.(*LockIdempotencyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_InitiateTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitiateTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).InitiateTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_InitiateTransaction_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).InitiateTransaction(ctx, req.(*InitiateTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BankService_ServiceDesc is the grpc.ServiceDesc for BankService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BankService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gw_bank.BankService",
	HandlerType: (*BankServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterClient",
			Handler:    _BankService_RegisterClient_Handler,
		},
		{
			MethodName: "UpdateClientDetails",
			Handler:    _BankService_UpdateClientDetails_Handler,
		},
		{
			MethodName: "ViewBalance",
			Handler:    _BankService_ViewBalance_Handler,
		},
		{
			MethodName: "LockIdempotency",
			Handler:    _BankService_LockIdempotency_Handler,
		},
		{
			MethodName: "InitiateTransaction",
			Handler:    _BankService_InitiateTransaction_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/gw-bank.proto",
}
