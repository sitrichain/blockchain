// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: peer/peer.proto

package peer

import (
	context "context"
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	proto "github.com/gogo/protobuf/proto"
	common "github.com/rongzer/blockchain/protos/common"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type PeerID struct {
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (m *PeerID) Reset()         { *m = PeerID{} }
func (m *PeerID) String() string { return proto.CompactTextString(m) }
func (*PeerID) ProtoMessage()    {}
func (*PeerID) Descriptor() ([]byte, []int) {
	return fileDescriptor_c302117fbb08ad42, []int{0}
}
func (m *PeerID) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PeerID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_PeerID.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *PeerID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PeerID.Merge(m, src)
}
func (m *PeerID) XXX_Size() int {
	return m.Size()
}
func (m *PeerID) XXX_DiscardUnknown() {
	xxx_messageInfo_PeerID.DiscardUnknown(m)
}

var xxx_messageInfo_PeerID proto.InternalMessageInfo

func (m *PeerID) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

type PeerEndpoint struct {
	Id      *PeerID `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Address string  `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
}

func (m *PeerEndpoint) Reset()         { *m = PeerEndpoint{} }
func (m *PeerEndpoint) String() string { return proto.CompactTextString(m) }
func (*PeerEndpoint) ProtoMessage()    {}
func (*PeerEndpoint) Descriptor() ([]byte, []int) {
	return fileDescriptor_c302117fbb08ad42, []int{1}
}
func (m *PeerEndpoint) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PeerEndpoint) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_PeerEndpoint.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *PeerEndpoint) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PeerEndpoint.Merge(m, src)
}
func (m *PeerEndpoint) XXX_Size() int {
	return m.Size()
}
func (m *PeerEndpoint) XXX_DiscardUnknown() {
	xxx_messageInfo_PeerEndpoint.DiscardUnknown(m)
}

var xxx_messageInfo_PeerEndpoint proto.InternalMessageInfo

func (m *PeerEndpoint) GetId() *PeerID {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *PeerEndpoint) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func init() {
	proto.RegisterType((*PeerID)(nil), "protos.PeerID")
	proto.RegisterType((*PeerEndpoint)(nil), "protos.PeerEndpoint")
}

func init() { proto.RegisterFile("peer/peer.proto", fileDescriptor_c302117fbb08ad42) }

var fileDescriptor_c302117fbb08ad42 = []byte{
	// 320 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0xb1, 0x4e, 0xeb, 0x30,
	0x14, 0x86, 0x93, 0xea, 0xaa, 0xf7, 0x5e, 0x83, 0x28, 0x32, 0x12, 0x8a, 0xaa, 0xca, 0x42, 0x99,
	0x60, 0x49, 0xa4, 0xf2, 0x04, 0x14, 0x2a, 0xc1, 0x80, 0x54, 0xa5, 0x4c, 0x2c, 0xc8, 0xb1, 0x8f,
	0x52, 0x8b, 0xc6, 0x27, 0xf2, 0x09, 0x0b, 0x4f, 0xc1, 0xc0, 0x43, 0x31, 0x76, 0x64, 0x44, 0xed,
	0x8b, 0xa0, 0xc4, 0x09, 0x88, 0x85, 0x25, 0xb1, 0xff, 0xff, 0xf7, 0x77, 0x7c, 0x7c, 0xd8, 0xa8,
	0x02, 0x70, 0x69, 0xf3, 0x49, 0x2a, 0x87, 0x35, 0xf2, 0x61, 0xfb, 0xa3, 0xf1, 0x91, 0x37, 0x1c,
	0x56, 0x48, 0x72, 0xed, 0xcd, 0xf1, 0xe4, 0x87, 0xf8, 0xe0, 0x80, 0x2a, 0xb4, 0x04, 0x9d, 0x7b,
	0xa8, 0xb0, 0x2c, 0xd1, 0xa6, 0x2e, 0x57, 0x5e, 0x89, 0x27, 0x6c, 0xb8, 0x00, 0x70, 0x37, 0x57,
	0x9c, 0xb3, 0x3f, 0x56, 0x96, 0x10, 0x85, 0x27, 0xe1, 0xe9, 0xff, 0xac, 0x5d, 0xc7, 0xd7, 0x6c,
	0xbf, 0x71, 0xe7, 0x56, 0x57, 0x68, 0x6c, 0xcd, 0x05, 0x1b, 0x18, 0xdd, 0x26, 0xf6, 0xa6, 0x07,
	0x9e, 0x40, 0x89, 0x3f, 0x9f, 0x0d, 0x8c, 0xe6, 0x11, 0xfb, 0x2b, 0xb5, 0x76, 0x40, 0x14, 0x0d,
	0x5a, 0x4c, 0xbf, 0x9d, 0xbe, 0x86, 0xec, 0xdf, 0xdc, 0x6a, 0x74, 0x04, 0x8e, 0xcf, 0xd9, 0x68,
	0xe1, 0x50, 0x01, 0xd1, 0xa2, 0xbb, 0x28, 0x3f, 0xee, 0x69, 0x4b, 0x53, 0x58, 0xd0, 0xbd, 0x3e,
	0x8e, 0xbe, 0xaa, 0x74, 0x4a, 0xd6, 0x75, 0x14, 0x07, 0xfc, 0x82, 0x8d, 0x96, 0x60, 0xf5, 0x9d,
	0x93, 0x96, 0xa4, 0xaa, 0x0d, 0x5a, 0xce, 0x13, 0xdf, 0x61, 0x92, 0xcd, 0x2e, 0x6f, 0x81, 0x48,
	0x16, 0xf0, 0x1b, 0x62, 0x56, 0xbc, 0x6d, 0x45, 0xb8, 0xd9, 0x8a, 0xf0, 0x63, 0x2b, 0xc2, 0x97,
	0x9d, 0x08, 0x36, 0x3b, 0x11, 0xbc, 0xef, 0x44, 0xc0, 0x62, 0x85, 0x65, 0xe2, 0xd0, 0x16, 0xcf,
	0xe0, 0x92, 0x7c, 0x8d, 0xea, 0x51, 0xad, 0xa4, 0xb1, 0x3d, 0xab, 0x79, 0xe6, 0xfb, 0xb3, 0xc2,
	0xd4, 0xab, 0xa7, 0xbc, 0xa9, 0x99, 0x76, 0xd1, 0xf4, 0x3b, 0x9a, 0xfa, 0x68, 0x3b, 0xba, 0xdc,
	0x0f, 0xed, 0xfc, 0x33, 0x00, 0x00, 0xff, 0xff, 0x83, 0xcc, 0xef, 0x5e, 0xce, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// EndorserClient is the client API for Endorser service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type EndorserClient interface {
	ProcessProposal(ctx context.Context, in *SignedProposal, opts ...grpc.CallOption) (*ProposalResponse, error)
	SendTransaction(ctx context.Context, in *common.RBCMessage, opts ...grpc.CallOption) (*ProposalResponse, error)
}

type endorserClient struct {
	cc *grpc.ClientConn
}

func NewEndorserClient(cc *grpc.ClientConn) EndorserClient {
	return &endorserClient{cc}
}

func (c *endorserClient) ProcessProposal(ctx context.Context, in *SignedProposal, opts ...grpc.CallOption) (*ProposalResponse, error) {
	out := new(ProposalResponse)
	err := c.cc.Invoke(ctx, "/protos.Endorser/ProcessProposal", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *endorserClient) SendTransaction(ctx context.Context, in *common.RBCMessage, opts ...grpc.CallOption) (*ProposalResponse, error) {
	out := new(ProposalResponse)
	err := c.cc.Invoke(ctx, "/protos.Endorser/SendTransaction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EndorserServer is the server API for Endorser service.
type EndorserServer interface {
	ProcessProposal(context.Context, *SignedProposal) (*ProposalResponse, error)
	SendTransaction(context.Context, *common.RBCMessage) (*ProposalResponse, error)
}

// UnimplementedEndorserServer can be embedded to have forward compatible implementations.
type UnimplementedEndorserServer struct {
}

func (*UnimplementedEndorserServer) ProcessProposal(ctx context.Context, req *SignedProposal) (*ProposalResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProcessProposal not implemented")
}
func (*UnimplementedEndorserServer) SendTransaction(ctx context.Context, req *common.RBCMessage) (*ProposalResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendTransaction not implemented")
}

func RegisterEndorserServer(s *grpc.Server, srv EndorserServer) {
	s.RegisterService(&_Endorser_serviceDesc, srv)
}

func _Endorser_ProcessProposal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignedProposal)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EndorserServer).ProcessProposal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Endorser/ProcessProposal",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EndorserServer).ProcessProposal(ctx, req.(*SignedProposal))
	}
	return interceptor(ctx, in, info, handler)
}

func _Endorser_SendTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(common.RBCMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EndorserServer).SendTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Endorser/SendTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EndorserServer).SendTransaction(ctx, req.(*common.RBCMessage))
	}
	return interceptor(ctx, in, info, handler)
}

var _Endorser_serviceDesc = grpc.ServiceDesc{
	ServiceName: "protos.Endorser",
	HandlerType: (*EndorserServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ProcessProposal",
			Handler:    _Endorser_ProcessProposal_Handler,
		},
		{
			MethodName: "SendTransaction",
			Handler:    _Endorser_SendTransaction_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "peer/peer.proto",
}

func (m *PeerID) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PeerID) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PeerID) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintPeer(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *PeerEndpoint) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PeerEndpoint) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PeerEndpoint) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintPeer(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0x12
	}
	if m.Id != nil {
		{
			size, err := m.Id.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintPeer(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintPeer(dAtA []byte, offset int, v uint64) int {
	offset -= sovPeer(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *PeerID) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovPeer(uint64(l))
	}
	return n
}

func (m *PeerEndpoint) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Id != nil {
		l = m.Id.Size()
		n += 1 + l + sovPeer(uint64(l))
	}
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovPeer(uint64(l))
	}
	return n
}

func sovPeer(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPeer(x uint64) (n int) {
	return sovPeer(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *PeerID) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPeer
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PeerID: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PeerID: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPeer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthPeer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPeer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPeer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPeer
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPeer
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *PeerEndpoint) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPeer
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PeerEndpoint: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PeerEndpoint: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPeer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthPeer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPeer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Id == nil {
				m.Id = &PeerID{}
			}
			if err := m.Id.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPeer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthPeer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPeer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPeer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPeer
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPeer
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipPeer(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPeer
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowPeer
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowPeer
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthPeer
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthPeer
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowPeer
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipPeer(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthPeer
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthPeer = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPeer   = fmt.Errorf("proto: integer overflow")
)