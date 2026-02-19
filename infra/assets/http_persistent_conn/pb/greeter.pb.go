package pb

import (
    context "context"
    reflect "reflect"
    sync "sync"

    grpc "google.golang.org/grpc"
    codes "google.golang.org/grpc/codes"
    status "google.golang.org/grpc/status"
    protoreflect "google.golang.org/protobuf/reflect/protoreflect"
    protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const _ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
const _ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)

// HelloRequest の定義
// Deprecated: Use HelloRequest.ProtoReflect.Descriptor instead.
func (*HelloRequest) Descriptor() ([]byte, []int) {
    return file_proto_greeter_proto_rawDescGZIP(), []int{0}
}

func (x *HelloRequest) Reset() {
    *x = HelloRequest{}
    if protoimpl.UnsafeEnabled {
        mi := &file_proto_greeter_proto_msgTypes[0]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *HelloRequest) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*HelloRequest) ProtoMessage() {}

func (x *HelloRequest) ProtoReflect() protoreflect.Message {
    mi := &file_proto_greeter_proto_msgTypes[0]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

func (x *HelloRequest) GetName() string {
    if x != nil {
        return x.Name
    }
    return ""
}

type HelloReply struct {
    state protoimpl.MessageState
    sizeCache protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

func (*HelloReply) Descriptor() ([]byte, []int) {
    return file_proto_greeter_proto_rawDescGZIP(), []int{1}
}

func (x *HelloReply) Reset() {
    *x = HelloReply{}
    if protoimpl.UnsafeEnabled {
        mi := &file_proto_greeter_proto_msgTypes[1]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *HelloReply) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*HelloReply) ProtoMessage() {}

func (x *HelloReply) ProtoReflect() protoreflect.Message {
    mi := &file_proto_greeter_proto_msgTypes[1]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

func (x *HelloReply) GetMessage() string {
    if x != nil {
        return x.Message
    }
    return ""
}

type GreeterClient interface {
    SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error)
    SayHelloStream(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (Greeter_SayHelloStreamClient, error)
}

type greeterClient struct {
    cc grpc.ClientConnInterface
}

func NewGreeterClient(cc grpc.ClientConnInterface) GreeterClient {
    return &greeterClient{cc}
}

func (c *greeterClient) SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
    out := new(HelloReply)
    err := c.cc.Invoke(ctx, "/pb.Greeter/SayHello", in, out, opts...)
    if err != nil {
        return nil, err
    }
    return out, nil
}

func (c *greeterClient) SayHelloStream(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (Greeter_SayHelloStreamClient, error) {
    stream, err := c.cc.NewStream(ctx, &grpc.StreamDesc{StreamName: "SayHelloStream", ServerStreams: true}, "/pb.Greeter/SayHelloStream", opts...)
    if err != nil {
        return nil, err
    }
    x := &greeterSayHelloStreamClient{stream}
    if err := x.ClientStream.SendMsg(in); err != nil {
        return nil, err
    }
    if err := x.ClientStream.CloseSend(); err != nil {
        return nil, err
    }
    return x, nil
}

type Greeter_SayHelloStreamClient interface {
    Recv() (*HelloReply, error)
    grpc.ClientStream
}

type greeterSayHelloStreamClient struct {
    grpc.ClientStream
}

func (x *greeterSayHelloStreamClient) Recv() (*HelloReply, error) {
    m := new(HelloReply)
    if err := x.ClientStream.RecvMsg(m); err != nil {
        return nil, err
    }
    return m, nil
}

type GreeterServer interface {
    SayHello(context.Context, *HelloRequest) (*HelloReply, error)
    SayHelloStream(*HelloRequest, Greeter_SayHelloStreamServer) error
    mustEmbedUnimplementedGreeterServer()
}

type UnimplementedGreeterServer struct {}

func (UnimplementedGreeterServer) SayHello(context.Context, *HelloRequest) (*HelloReply, error) {
    return nil, status.Errorf(codes.Unimplemented, "method SayHello not implemented")
}

func (UnimplementedGreeterServer) SayHelloStream(*HelloRequest, Greeter_SayHelloStreamServer) error {
    return status.Errorf(codes.Unimplemented, "method SayHelloStream not implemented")
}

func (UnimplementedGreeterServer) mustEmbedUnimplementedGreeterServer() {}

type UnsafeGreeterServer interface {
    mustEmbedUnimplementedGreeterServer()
}

func RegisterGreeterServer(s grpc.ServiceRegistrar, srv GreeterServer) {
    s.RegisterService(&grpc.ServiceDesc{
        ServiceName: "pb.Greeter",
        HandlerType: (*GreeterServer)(nil),
        Methods: []grpc.MethodDesc{
            {MethodName: "SayHello", Handler: _Greeter_SayHello_Handler},
        },
        Streams: []grpc.StreamDesc{
            {StreamName: "SayHelloStream", Handler: _Greeter_SayHelloStream_Handler, ServerStreams: true},
        },
        Metadata: "proto/greeter.proto",
    }, srv)
}

func _Greeter_SayHello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
    in := new(HelloRequest)
    if err := dec(in); err != nil {
        return nil, err
    }
    if interceptor == nil {
        return srv.(GreeterServer).SayHello(ctx, in)
    }
    info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pb.Greeter/SayHello"}
    handler := func(ctx context.Context, req interface{}) (interface{}, error) {
        return srv.(GreeterServer).SayHello(ctx, req.(*HelloRequest))
    }
    return interceptor(ctx, in, info, handler)
}

func _Greeter_SayHelloStream_Handler(srv interface{}, stream grpc.ServerStream) error {
    m := new(HelloRequest)
    if err := stream.RecvMsg(m); err != nil {
        return err
    }
    return srv.(GreeterServer).SayHelloStream(m, &greeterSayHelloStreamServer{stream})
}

type Greeter_SayHelloStreamServer interface {
    Send(*HelloReply) error
    grpc.ServerStream
}

type greeterSayHelloStreamServer struct {
    grpc.ServerStream
}

func (x *greeterSayHelloStreamServer) Send(m *HelloReply) error {
    return x.ServerStream.SendMsg(m)
}

var File_proto_greeter_proto protoreflect.FileDescriptor

var file_proto_greeter_proto_rawDesc = []byte{
  0x0a, 0xa0, 0x02, 0x0a, 0x13, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67,
  0x72, 0x65, 0x65, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
  0x12, 0x02, 0x70, 0x62, 0x22, 0x22, 0x0a, 0x0c, 0x48, 0x65, 0x6c, 0x6c,
  0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04,
  0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
  0x6e, 0x61, 0x6d, 0x65, 0x22, 0x26, 0x0a, 0x0a, 0x48, 0x65, 0x6c, 0x6c,
  0x6f, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65,
  0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
  0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0x6d, 0x0a, 0x07,
  0x47, 0x72, 0x65, 0x65, 0x74, 0x65, 0x72, 0x12, 0x2c, 0x0a, 0x08, 0x53,
  0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x12, 0x10, 0x2e, 0x70, 0x62,
  0x2e, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
  0x74, 0x1a, 0x0e, 0x2e, 0x70, 0x62, 0x2e, 0x48, 0x65, 0x6c, 0x6c, 0x6f,
  0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x34, 0x0a, 0x0e, 0x53, 0x61, 0x79,
  0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12,
  0x10, 0x2e, 0x70, 0x62, 0x2e, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x52, 0x65,
  0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0e, 0x2e, 0x70, 0x62, 0x2e, 0x48,
  0x65, 0x6c, 0x6c, 0x6f, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x30, 0x01, 0x42,
  0x42, 0x5a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
  0x6d, 0x2f, 0x73, 0x6f, 0x6b, 0x6f, 0x69, 0x64, 0x65, 0x2f, 0x77, 0x6f,
  0x72, 0x6b, 0x73, 0x68, 0x6f, 0x70, 0x2f, 0x69, 0x6e, 0x66, 0x72, 0x61,
  0x2f, 0x61, 0x73, 0x73, 0x65, 0x74, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70,
  0x5f, 0x70, 0x65, 0x72, 0x73, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x74, 0x5f,
  0x63, 0x6f, 0x6e, 0x6e, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f,
  0x74, 0x6f, 0x33,
}

var (
    file_proto_greeter_proto_rawDescOnce sync.Once
    file_proto_greeter_proto_rawDescData = file_proto_greeter_proto_rawDesc
)

func file_proto_greeter_proto_rawDescGZIP() []byte {
    file_proto_greeter_proto_rawDescOnce.Do(func() {
        file_proto_greeter_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_greeter_proto_rawDescData)
    })
    return file_proto_greeter_proto_rawDescData
}

var file_proto_greeter_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_proto_greeter_proto_goTypes = []interface{}{
    (*HelloRequest)(nil), // 0: pb.HelloRequest
    (*HelloReply)(nil), // 1: pb.HelloReply
}
var file_proto_greeter_proto_depIdxs = []int32{
    0, // 0: pb.Greeter.SayHello:input_type -> pb.HelloRequest
    1, // 1: pb.Greeter.SayHello:output_type -> pb.HelloReply
    0, // 2: pb.Greeter.SayHelloStream:input_type -> pb.HelloRequest
    1, // 3: pb.Greeter.SayHelloStream:output_type -> pb.HelloReply
}

func init() {
    if File_proto_greeter_proto != nil {
        return
    }
    if !protoimpl.UnsafeEnabled {
        file_proto_greeter_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*HelloRequest); i {
            case 0:
                return &v.state
            case 1:
                return &v.sizeCache
            case 2:
                return &v.unknownFields
            default:
                return nil
            }
        }
        file_proto_greeter_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*HelloReply); i {
            case 0:
                return &v.state
            case 1:
                return &v.sizeCache
            case 2:
                return &v.unknownFields
            default:
                return nil
            }
        }
    }
    type x struct {}
    out := protoimpl.TypeBuilder{
        File: protoimpl.DescBuilder{
            GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
            RawDescriptor: file_proto_greeter_proto_rawDesc,
            NumEnums: 0,
            NumMessages: 2,
            NumExtensions: 0,
            NumServices: 1,
        },
        GoTypes: file_proto_greeter_proto_goTypes,
        DependencyIndexes: file_proto_greeter_proto_depIdxs,
        MessageInfos: file_proto_greeter_proto_msgTypes,
    }.Build()
    File_proto_greeter_proto = out.File
    file_proto_greeter_proto_rawDesc = nil
    file_proto_greeter_proto_goTypes = nil
    file_proto_greeter_proto_depIdxs = nil
}
