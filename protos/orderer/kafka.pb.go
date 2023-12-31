// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: orderer/kafka.proto

package orderer

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
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

// KafkaMessage is a wrapper type for the messages
// that the Kafka-based orderer deals with.
type KafkaMessage struct {
	// Types that are valid to be assigned to Type:
	//	*KafkaMessage_Regular
	//	*KafkaMessage_TimeToCut
	//	*KafkaMessage_Connect
	Type isKafkaMessage_Type `protobuf_oneof:"Type"`
}

func (m *KafkaMessage) Reset()         { *m = KafkaMessage{} }
func (m *KafkaMessage) String() string { return proto.CompactTextString(m) }
func (*KafkaMessage) ProtoMessage()    {}
func (*KafkaMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_44ca1a6a801d36ec, []int{0}
}
func (m *KafkaMessage) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *KafkaMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_KafkaMessage.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *KafkaMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KafkaMessage.Merge(m, src)
}
func (m *KafkaMessage) XXX_Size() int {
	return m.Size()
}
func (m *KafkaMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_KafkaMessage.DiscardUnknown(m)
}

var xxx_messageInfo_KafkaMessage proto.InternalMessageInfo

type isKafkaMessage_Type interface {
	isKafkaMessage_Type()
	MarshalTo([]byte) (int, error)
	Size() int
}

type KafkaMessage_Regular struct {
	Regular *KafkaMessageRegular `protobuf:"bytes,1,opt,name=regular,proto3,oneof" json:"regular,omitempty"`
}
type KafkaMessage_TimeToCut struct {
	TimeToCut *KafkaMessageTimeToCut `protobuf:"bytes,2,opt,name=time_to_cut,json=timeToCut,proto3,oneof" json:"time_to_cut,omitempty"`
}
type KafkaMessage_Connect struct {
	Connect *KafkaMessageConnect `protobuf:"bytes,3,opt,name=connect,proto3,oneof" json:"connect,omitempty"`
}

func (*KafkaMessage_Regular) isKafkaMessage_Type()   {}
func (*KafkaMessage_TimeToCut) isKafkaMessage_Type() {}
func (*KafkaMessage_Connect) isKafkaMessage_Type()   {}

func (m *KafkaMessage) GetType() isKafkaMessage_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *KafkaMessage) GetRegular() *KafkaMessageRegular {
	if x, ok := m.GetType().(*KafkaMessage_Regular); ok {
		return x.Regular
	}
	return nil
}

func (m *KafkaMessage) GetTimeToCut() *KafkaMessageTimeToCut {
	if x, ok := m.GetType().(*KafkaMessage_TimeToCut); ok {
		return x.TimeToCut
	}
	return nil
}

func (m *KafkaMessage) GetConnect() *KafkaMessageConnect {
	if x, ok := m.GetType().(*KafkaMessage_Connect); ok {
		return x.Connect
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*KafkaMessage) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*KafkaMessage_Regular)(nil),
		(*KafkaMessage_TimeToCut)(nil),
		(*KafkaMessage_Connect)(nil),
	}
}

// KafkaMessageRegular wraps a marshalled envelope.
type KafkaMessageRegular struct {
	Payload []byte `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (m *KafkaMessageRegular) Reset()         { *m = KafkaMessageRegular{} }
func (m *KafkaMessageRegular) String() string { return proto.CompactTextString(m) }
func (*KafkaMessageRegular) ProtoMessage()    {}
func (*KafkaMessageRegular) Descriptor() ([]byte, []int) {
	return fileDescriptor_44ca1a6a801d36ec, []int{1}
}
func (m *KafkaMessageRegular) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *KafkaMessageRegular) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_KafkaMessageRegular.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *KafkaMessageRegular) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KafkaMessageRegular.Merge(m, src)
}
func (m *KafkaMessageRegular) XXX_Size() int {
	return m.Size()
}
func (m *KafkaMessageRegular) XXX_DiscardUnknown() {
	xxx_messageInfo_KafkaMessageRegular.DiscardUnknown(m)
}

var xxx_messageInfo_KafkaMessageRegular proto.InternalMessageInfo

func (m *KafkaMessageRegular) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

// KafkaMessageTimeToCut is used to signal to the orderers
// that it is time to cut block <block_number>.
type KafkaMessageTimeToCut struct {
	BlockNumber uint64 `protobuf:"varint,1,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
}

func (m *KafkaMessageTimeToCut) Reset()         { *m = KafkaMessageTimeToCut{} }
func (m *KafkaMessageTimeToCut) String() string { return proto.CompactTextString(m) }
func (*KafkaMessageTimeToCut) ProtoMessage()    {}
func (*KafkaMessageTimeToCut) Descriptor() ([]byte, []int) {
	return fileDescriptor_44ca1a6a801d36ec, []int{2}
}
func (m *KafkaMessageTimeToCut) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *KafkaMessageTimeToCut) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_KafkaMessageTimeToCut.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *KafkaMessageTimeToCut) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KafkaMessageTimeToCut.Merge(m, src)
}
func (m *KafkaMessageTimeToCut) XXX_Size() int {
	return m.Size()
}
func (m *KafkaMessageTimeToCut) XXX_DiscardUnknown() {
	xxx_messageInfo_KafkaMessageTimeToCut.DiscardUnknown(m)
}

var xxx_messageInfo_KafkaMessageTimeToCut proto.InternalMessageInfo

func (m *KafkaMessageTimeToCut) GetBlockNumber() uint64 {
	if m != nil {
		return m.BlockNumber
	}
	return 0
}

// KafkaMessageConnect is posted by an orderer upon booting up.
// It is used to prevent the panic that would be caused if we
// were to consume an empty partition. It is ignored by all
// orderers when processing the partition.
type KafkaMessageConnect struct {
	Payload []byte `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (m *KafkaMessageConnect) Reset()         { *m = KafkaMessageConnect{} }
func (m *KafkaMessageConnect) String() string { return proto.CompactTextString(m) }
func (*KafkaMessageConnect) ProtoMessage()    {}
func (*KafkaMessageConnect) Descriptor() ([]byte, []int) {
	return fileDescriptor_44ca1a6a801d36ec, []int{3}
}
func (m *KafkaMessageConnect) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *KafkaMessageConnect) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_KafkaMessageConnect.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *KafkaMessageConnect) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KafkaMessageConnect.Merge(m, src)
}
func (m *KafkaMessageConnect) XXX_Size() int {
	return m.Size()
}
func (m *KafkaMessageConnect) XXX_DiscardUnknown() {
	xxx_messageInfo_KafkaMessageConnect.DiscardUnknown(m)
}

var xxx_messageInfo_KafkaMessageConnect proto.InternalMessageInfo

func (m *KafkaMessageConnect) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

// LastOffsetPersisted is the encoded value for the Metadata message
// which is encoded in the ORDERER block metadata index for the case
// of the Kafka-based orderer.
type KafkaMetadata struct {
	LastOffsetPersisted int64 `protobuf:"varint,1,opt,name=last_offset_persisted,json=lastOffsetPersisted,proto3" json:"last_offset_persisted,omitempty"`
}

func (m *KafkaMetadata) Reset()         { *m = KafkaMetadata{} }
func (m *KafkaMetadata) String() string { return proto.CompactTextString(m) }
func (*KafkaMetadata) ProtoMessage()    {}
func (*KafkaMetadata) Descriptor() ([]byte, []int) {
	return fileDescriptor_44ca1a6a801d36ec, []int{4}
}
func (m *KafkaMetadata) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *KafkaMetadata) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_KafkaMetadata.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *KafkaMetadata) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KafkaMetadata.Merge(m, src)
}
func (m *KafkaMetadata) XXX_Size() int {
	return m.Size()
}
func (m *KafkaMetadata) XXX_DiscardUnknown() {
	xxx_messageInfo_KafkaMetadata.DiscardUnknown(m)
}

var xxx_messageInfo_KafkaMetadata proto.InternalMessageInfo

func (m *KafkaMetadata) GetLastOffsetPersisted() int64 {
	if m != nil {
		return m.LastOffsetPersisted
	}
	return 0
}

func init() {
	proto.RegisterType((*KafkaMessage)(nil), "orderer.KafkaMessage")
	proto.RegisterType((*KafkaMessageRegular)(nil), "orderer.KafkaMessageRegular")
	proto.RegisterType((*KafkaMessageTimeToCut)(nil), "orderer.KafkaMessageTimeToCut")
	proto.RegisterType((*KafkaMessageConnect)(nil), "orderer.KafkaMessageConnect")
	proto.RegisterType((*KafkaMetadata)(nil), "orderer.KafkaMetadata")
}

func init() { proto.RegisterFile("orderer/kafka.proto", fileDescriptor_44ca1a6a801d36ec) }

var fileDescriptor_44ca1a6a801d36ec = []byte{
	// 345 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0xbf, 0x6a, 0xf3, 0x30,
	0x14, 0xc5, 0xed, 0x2f, 0x21, 0xe1, 0x53, 0xd2, 0xc5, 0x21, 0xe0, 0xa1, 0x88, 0x36, 0x50, 0xe8,
	0x50, 0x6c, 0x48, 0x97, 0xd2, 0xa9, 0x24, 0x4b, 0xa0, 0xf4, 0x0f, 0x26, 0x53, 0x17, 0x23, 0x2b,
	0x37, 0x8e, 0x89, 0x6d, 0x19, 0xe9, 0x7a, 0x48, 0x9f, 0xa2, 0x8f, 0xd5, 0xa1, 0x43, 0xc6, 0x8e,
	0x25, 0x79, 0x91, 0x62, 0x59, 0xa6, 0xa5, 0x84, 0x8c, 0xf7, 0x9e, 0xdf, 0xe1, 0x1c, 0x5d, 0x91,
	0x81, 0x90, 0x0b, 0x90, 0x20, 0xfd, 0x35, 0x5b, 0xae, 0x99, 0x57, 0x48, 0x81, 0xc2, 0xe9, 0x9a,
	0xe5, 0xe8, 0xc3, 0x26, 0xfd, 0xfb, 0x4a, 0x78, 0x00, 0xa5, 0x58, 0x0c, 0xce, 0x0d, 0xe9, 0x4a,
	0x88, 0xcb, 0x94, 0x49, 0xd7, 0x3e, 0xb3, 0x2f, 0x7b, 0xe3, 0x53, 0xcf, 0xb0, 0xde, 0x6f, 0x2e,
	0xa8, 0x99, 0x99, 0x15, 0x34, 0xb8, 0x73, 0x47, 0x7a, 0x98, 0x64, 0x10, 0xa2, 0x08, 0x79, 0x89,
	0xee, 0x3f, 0xed, 0xa6, 0x07, 0xdd, 0xf3, 0x24, 0x83, 0xb9, 0x98, 0x96, 0x38, 0xb3, 0x82, 0xff,
	0xd8, 0x0c, 0x55, 0x36, 0x17, 0x79, 0x0e, 0x1c, 0xdd, 0xd6, 0x91, 0xec, 0x69, 0xcd, 0x54, 0xd9,
	0x06, 0x9f, 0x74, 0x48, 0x7b, 0xbe, 0x29, 0x60, 0xe4, 0x93, 0xc1, 0x81, 0x96, 0x8e, 0x4b, 0xba,
	0x05, 0xdb, 0xa4, 0x82, 0x2d, 0xf4, 0xa3, 0xfa, 0x41, 0x33, 0x8e, 0x6e, 0xc9, 0xf0, 0x60, 0x31,
	0xe7, 0x9c, 0xf4, 0xa3, 0x54, 0xf0, 0x75, 0x98, 0x97, 0x59, 0x04, 0xf5, 0x31, 0xda, 0x41, 0x4f,
	0xef, 0x1e, 0xf5, 0xea, 0x6f, 0x98, 0xa9, 0x75, 0x24, 0x6c, 0x4a, 0x4e, 0x8c, 0x01, 0xd9, 0x82,
	0x21, 0x73, 0xc6, 0x64, 0x98, 0x32, 0x85, 0xa1, 0x58, 0x2e, 0x15, 0x60, 0x58, 0x80, 0x54, 0x89,
	0x42, 0xa8, 0x8d, 0xad, 0x60, 0x50, 0x89, 0x4f, 0x5a, 0x7b, 0x6e, 0xa4, 0x49, 0xf6, 0xbe, 0xa3,
	0xf6, 0x76, 0x47, 0xed, 0xaf, 0x1d, 0xb5, 0xdf, 0xf6, 0xd4, 0xda, 0xee, 0xa9, 0xf5, 0xb9, 0xa7,
	0x16, 0xb9, 0xe0, 0x22, 0xf3, 0xa4, 0xc8, 0xe3, 0x57, 0x90, 0x9e, 0x2e, 0xca, 0x57, 0x2c, 0xc9,
	0xeb, 0x2f, 0x57, 0xcd, 0x29, 0x5f, 0xae, 0xe2, 0x04, 0x57, 0x65, 0xe4, 0x71, 0x91, 0xf9, 0x86,
	0xf6, 0x7f, 0x68, 0xbf, 0xa6, 0x7d, 0x43, 0x47, 0x1d, 0x3d, 0x5f, 0x7f, 0x07, 0x00, 0x00, 0xff,
	0xff, 0xee, 0x64, 0x8e, 0xe3, 0x47, 0x02, 0x00, 0x00,
}

func (m *KafkaMessage) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *KafkaMessage) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KafkaMessage) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Type != nil {
		{
			size := m.Type.Size()
			i -= size
			if _, err := m.Type.MarshalTo(dAtA[i:]); err != nil {
				return 0, err
			}
		}
	}
	return len(dAtA) - i, nil
}

func (m *KafkaMessage_Regular) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KafkaMessage_Regular) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.Regular != nil {
		{
			size, err := m.Regular.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintKafka(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}
func (m *KafkaMessage_TimeToCut) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KafkaMessage_TimeToCut) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.TimeToCut != nil {
		{
			size, err := m.TimeToCut.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintKafka(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	return len(dAtA) - i, nil
}
func (m *KafkaMessage_Connect) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KafkaMessage_Connect) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.Connect != nil {
		{
			size, err := m.Connect.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintKafka(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	return len(dAtA) - i, nil
}
func (m *KafkaMessageRegular) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *KafkaMessageRegular) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KafkaMessageRegular) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Payload) > 0 {
		i -= len(m.Payload)
		copy(dAtA[i:], m.Payload)
		i = encodeVarintKafka(dAtA, i, uint64(len(m.Payload)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *KafkaMessageTimeToCut) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *KafkaMessageTimeToCut) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KafkaMessageTimeToCut) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.BlockNumber != 0 {
		i = encodeVarintKafka(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *KafkaMessageConnect) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *KafkaMessageConnect) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KafkaMessageConnect) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Payload) > 0 {
		i -= len(m.Payload)
		copy(dAtA[i:], m.Payload)
		i = encodeVarintKafka(dAtA, i, uint64(len(m.Payload)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *KafkaMetadata) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *KafkaMetadata) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KafkaMetadata) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.LastOffsetPersisted != 0 {
		i = encodeVarintKafka(dAtA, i, uint64(m.LastOffsetPersisted))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintKafka(dAtA []byte, offset int, v uint64) int {
	offset -= sovKafka(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *KafkaMessage) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Type != nil {
		n += m.Type.Size()
	}
	return n
}

func (m *KafkaMessage_Regular) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Regular != nil {
		l = m.Regular.Size()
		n += 1 + l + sovKafka(uint64(l))
	}
	return n
}
func (m *KafkaMessage_TimeToCut) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.TimeToCut != nil {
		l = m.TimeToCut.Size()
		n += 1 + l + sovKafka(uint64(l))
	}
	return n
}
func (m *KafkaMessage_Connect) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Connect != nil {
		l = m.Connect.Size()
		n += 1 + l + sovKafka(uint64(l))
	}
	return n
}
func (m *KafkaMessageRegular) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Payload)
	if l > 0 {
		n += 1 + l + sovKafka(uint64(l))
	}
	return n
}

func (m *KafkaMessageTimeToCut) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.BlockNumber != 0 {
		n += 1 + sovKafka(uint64(m.BlockNumber))
	}
	return n
}

func (m *KafkaMessageConnect) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Payload)
	if l > 0 {
		n += 1 + l + sovKafka(uint64(l))
	}
	return n
}

func (m *KafkaMetadata) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.LastOffsetPersisted != 0 {
		n += 1 + sovKafka(uint64(m.LastOffsetPersisted))
	}
	return n
}

func sovKafka(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozKafka(x uint64) (n int) {
	return sovKafka(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *KafkaMessage) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowKafka
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
			return fmt.Errorf("proto: KafkaMessage: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: KafkaMessage: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Regular", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowKafka
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
				return ErrInvalidLengthKafka
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthKafka
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			v := &KafkaMessageRegular{}
			if err := v.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			m.Type = &KafkaMessage_Regular{v}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TimeToCut", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowKafka
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
				return ErrInvalidLengthKafka
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthKafka
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			v := &KafkaMessageTimeToCut{}
			if err := v.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			m.Type = &KafkaMessage_TimeToCut{v}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Connect", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowKafka
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
				return ErrInvalidLengthKafka
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthKafka
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			v := &KafkaMessageConnect{}
			if err := v.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			m.Type = &KafkaMessage_Connect{v}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipKafka(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthKafka
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthKafka
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
func (m *KafkaMessageRegular) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowKafka
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
			return fmt.Errorf("proto: KafkaMessageRegular: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: KafkaMessageRegular: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Payload", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowKafka
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthKafka
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthKafka
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Payload = append(m.Payload[:0], dAtA[iNdEx:postIndex]...)
			if m.Payload == nil {
				m.Payload = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipKafka(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthKafka
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthKafka
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
func (m *KafkaMessageTimeToCut) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowKafka
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
			return fmt.Errorf("proto: KafkaMessageTimeToCut: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: KafkaMessageTimeToCut: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockNumber", wireType)
			}
			m.BlockNumber = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowKafka
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.BlockNumber |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipKafka(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthKafka
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthKafka
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
func (m *KafkaMessageConnect) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowKafka
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
			return fmt.Errorf("proto: KafkaMessageConnect: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: KafkaMessageConnect: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Payload", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowKafka
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthKafka
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthKafka
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Payload = append(m.Payload[:0], dAtA[iNdEx:postIndex]...)
			if m.Payload == nil {
				m.Payload = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipKafka(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthKafka
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthKafka
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
func (m *KafkaMetadata) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowKafka
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
			return fmt.Errorf("proto: KafkaMetadata: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: KafkaMetadata: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastOffsetPersisted", wireType)
			}
			m.LastOffsetPersisted = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowKafka
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LastOffsetPersisted |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipKafka(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthKafka
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthKafka
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
func skipKafka(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowKafka
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
					return 0, ErrIntOverflowKafka
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
					return 0, ErrIntOverflowKafka
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
				return 0, ErrInvalidLengthKafka
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthKafka
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowKafka
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
				next, err := skipKafka(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthKafka
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
	ErrInvalidLengthKafka = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowKafka   = fmt.Errorf("proto: integer overflow")
)
