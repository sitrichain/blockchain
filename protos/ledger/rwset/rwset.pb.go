// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ledger/rwset/rwset.proto

package rwset

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

type TxReadWriteSet_DataModel int32

const (
	TxReadWriteSet_KV TxReadWriteSet_DataModel = 0
)

var TxReadWriteSet_DataModel_name = map[int32]string{
	0: "KV",
}

var TxReadWriteSet_DataModel_value = map[string]int32{
	"KV": 0,
}

func (x TxReadWriteSet_DataModel) String() string {
	return proto.EnumName(TxReadWriteSet_DataModel_name, int32(x))
}

func (TxReadWriteSet_DataModel) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_794d00b812408f20, []int{0, 0}
}

// TxReadWriteSet encapsulates a read-write set for a transaction
// DataModel specifies the enum value of the data model
// ns_rwset field specifies a list of chaincode specific read-write set (one for each chaincode)
type TxReadWriteSet struct {
	DataModel TxReadWriteSet_DataModel `protobuf:"varint,1,opt,name=data_model,json=dataModel,proto3,enum=rwset.TxReadWriteSet_DataModel" json:"data_model,omitempty"`
	NsRwset   []*NsReadWriteSet        `protobuf:"bytes,2,rep,name=ns_rwset,json=nsRwset,proto3" json:"ns_rwset,omitempty"`
}

func (m *TxReadWriteSet) Reset()         { *m = TxReadWriteSet{} }
func (m *TxReadWriteSet) String() string { return proto.CompactTextString(m) }
func (*TxReadWriteSet) ProtoMessage()    {}
func (*TxReadWriteSet) Descriptor() ([]byte, []int) {
	return fileDescriptor_794d00b812408f20, []int{0}
}
func (m *TxReadWriteSet) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *TxReadWriteSet) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_TxReadWriteSet.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *TxReadWriteSet) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TxReadWriteSet.Merge(m, src)
}
func (m *TxReadWriteSet) XXX_Size() int {
	return m.Size()
}
func (m *TxReadWriteSet) XXX_DiscardUnknown() {
	xxx_messageInfo_TxReadWriteSet.DiscardUnknown(m)
}

var xxx_messageInfo_TxReadWriteSet proto.InternalMessageInfo

func (m *TxReadWriteSet) GetDataModel() TxReadWriteSet_DataModel {
	if m != nil {
		return m.DataModel
	}
	return TxReadWriteSet_KV
}

func (m *TxReadWriteSet) GetNsRwset() []*NsReadWriteSet {
	if m != nil {
		return m.NsRwset
	}
	return nil
}

// NsReadWriteSet encapsulates the read-write set for a chaincode
type NsReadWriteSet struct {
	Namespace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Rwset     []byte `protobuf:"bytes,2,opt,name=rwset,proto3" json:"rwset,omitempty"`
}

func (m *NsReadWriteSet) Reset()         { *m = NsReadWriteSet{} }
func (m *NsReadWriteSet) String() string { return proto.CompactTextString(m) }
func (*NsReadWriteSet) ProtoMessage()    {}
func (*NsReadWriteSet) Descriptor() ([]byte, []int) {
	return fileDescriptor_794d00b812408f20, []int{1}
}
func (m *NsReadWriteSet) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *NsReadWriteSet) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_NsReadWriteSet.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *NsReadWriteSet) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NsReadWriteSet.Merge(m, src)
}
func (m *NsReadWriteSet) XXX_Size() int {
	return m.Size()
}
func (m *NsReadWriteSet) XXX_DiscardUnknown() {
	xxx_messageInfo_NsReadWriteSet.DiscardUnknown(m)
}

var xxx_messageInfo_NsReadWriteSet proto.InternalMessageInfo

func (m *NsReadWriteSet) GetNamespace() string {
	if m != nil {
		return m.Namespace
	}
	return ""
}

func (m *NsReadWriteSet) GetRwset() []byte {
	if m != nil {
		return m.Rwset
	}
	return nil
}

func init() {
	proto.RegisterEnum("rwset.TxReadWriteSet_DataModel", TxReadWriteSet_DataModel_name, TxReadWriteSet_DataModel_value)
	proto.RegisterType((*TxReadWriteSet)(nil), "rwset.TxReadWriteSet")
	proto.RegisterType((*NsReadWriteSet)(nil), "rwset.NsReadWriteSet")
}

func init() { proto.RegisterFile("ledger/rwset/rwset.proto", fileDescriptor_794d00b812408f20) }

var fileDescriptor_794d00b812408f20 = []byte{
	// 279 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0xc8, 0x49, 0x4d, 0x49,
	0x4f, 0x2d, 0xd2, 0x2f, 0x2a, 0x2f, 0x4e, 0x2d, 0x81, 0x90, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9,
	0x42, 0xac, 0x60, 0x8e, 0xd2, 0x74, 0x46, 0x2e, 0xbe, 0x90, 0x8a, 0xa0, 0xd4, 0xc4, 0x94, 0xf0,
	0xa2, 0xcc, 0x92, 0xd4, 0xe0, 0xd4, 0x12, 0x21, 0x3b, 0x2e, 0xae, 0x94, 0xc4, 0x92, 0xc4, 0xf8,
	0xdc, 0xfc, 0x94, 0xd4, 0x1c, 0x09, 0x46, 0x05, 0x46, 0x0d, 0x3e, 0x23, 0x79, 0x3d, 0x88, 0x5e,
	0x54, 0xa5, 0x7a, 0x2e, 0x89, 0x25, 0x89, 0xbe, 0x20, 0x65, 0x41, 0x9c, 0x29, 0x30, 0xa6, 0x90,
	0x01, 0x17, 0x47, 0x5e, 0x71, 0x3c, 0x58, 0xbd, 0x04, 0x93, 0x02, 0xb3, 0x06, 0xb7, 0x91, 0x28,
	0x54, 0xb7, 0x5f, 0x31, 0xb2, 0xee, 0x20, 0xf6, 0xbc, 0xe2, 0x20, 0xb0, 0x23, 0x84, 0xb9, 0x38,
	0xe1, 0x26, 0x09, 0xb1, 0x71, 0x31, 0x79, 0x87, 0x09, 0x30, 0x28, 0xb9, 0x70, 0xf1, 0xa1, 0xaa,
	0x17, 0x92, 0xe1, 0xe2, 0xcc, 0x4b, 0xcc, 0x4d, 0x2d, 0x2e, 0x48, 0x4c, 0x4e, 0x05, 0xbb, 0x8b,
	0x33, 0x08, 0x21, 0x20, 0x24, 0xc2, 0xc5, 0x0a, 0xb3, 0x93, 0x51, 0x83, 0x27, 0x08, 0xc2, 0x71,
	0x2a, 0x3f, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07, 0x8f, 0xe4, 0x18, 0x27, 0x3c,
	0x96, 0x63, 0xb8, 0xf0, 0x58, 0x8e, 0xe1, 0xc6, 0x63, 0x39, 0x06, 0x2e, 0xad, 0xe4, 0xfc, 0x5c,
	0xbd, 0xa2, 0xfc, 0xbc, 0xf4, 0xaa, 0xd4, 0x22, 0xbd, 0xa4, 0x9c, 0xfc, 0xe4, 0xec, 0xe4, 0x8c,
	0xc4, 0xcc, 0x3c, 0x48, 0xf0, 0x14, 0xeb, 0x41, 0x02, 0x0e, 0xe2, 0xf0, 0x28, 0xc3, 0xf4, 0xcc,
	0x92, 0x8c, 0xd2, 0x24, 0xbd, 0xe4, 0xfc, 0x5c, 0x7d, 0xa8, 0x16, 0x7d, 0x84, 0x16, 0x7d, 0x88,
	0x16, 0x7d, 0xe4, 0xb0, 0x4e, 0x62, 0x03, 0x0b, 0x1a, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x3b,
	0x16, 0xea, 0x0a, 0x82, 0x01, 0x00, 0x00,
}

func (m *TxReadWriteSet) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *TxReadWriteSet) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *TxReadWriteSet) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.NsRwset) > 0 {
		for iNdEx := len(m.NsRwset) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.NsRwset[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintRwset(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if m.DataModel != 0 {
		i = encodeVarintRwset(dAtA, i, uint64(m.DataModel))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *NsReadWriteSet) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *NsReadWriteSet) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *NsReadWriteSet) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Rwset) > 0 {
		i -= len(m.Rwset)
		copy(dAtA[i:], m.Rwset)
		i = encodeVarintRwset(dAtA, i, uint64(len(m.Rwset)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Namespace) > 0 {
		i -= len(m.Namespace)
		copy(dAtA[i:], m.Namespace)
		i = encodeVarintRwset(dAtA, i, uint64(len(m.Namespace)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintRwset(dAtA []byte, offset int, v uint64) int {
	offset -= sovRwset(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *TxReadWriteSet) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.DataModel != 0 {
		n += 1 + sovRwset(uint64(m.DataModel))
	}
	if len(m.NsRwset) > 0 {
		for _, e := range m.NsRwset {
			l = e.Size()
			n += 1 + l + sovRwset(uint64(l))
		}
	}
	return n
}

func (m *NsReadWriteSet) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Namespace)
	if l > 0 {
		n += 1 + l + sovRwset(uint64(l))
	}
	l = len(m.Rwset)
	if l > 0 {
		n += 1 + l + sovRwset(uint64(l))
	}
	return n
}

func sovRwset(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozRwset(x uint64) (n int) {
	return sovRwset(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *TxReadWriteSet) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRwset
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
			return fmt.Errorf("proto: TxReadWriteSet: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: TxReadWriteSet: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DataModel", wireType)
			}
			m.DataModel = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRwset
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.DataModel |= TxReadWriteSet_DataModel(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NsRwset", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRwset
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
				return ErrInvalidLengthRwset
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthRwset
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.NsRwset = append(m.NsRwset, &NsReadWriteSet{})
			if err := m.NsRwset[len(m.NsRwset)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRwset(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRwset
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthRwset
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
func (m *NsReadWriteSet) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRwset
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
			return fmt.Errorf("proto: NsReadWriteSet: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NsReadWriteSet: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Namespace", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRwset
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
				return ErrInvalidLengthRwset
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRwset
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Namespace = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rwset", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRwset
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
				return ErrInvalidLengthRwset
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRwset
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Rwset = append(m.Rwset[:0], dAtA[iNdEx:postIndex]...)
			if m.Rwset == nil {
				m.Rwset = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRwset(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRwset
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthRwset
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
func skipRwset(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowRwset
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
					return 0, ErrIntOverflowRwset
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
					return 0, ErrIntOverflowRwset
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
				return 0, ErrInvalidLengthRwset
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthRwset
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowRwset
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
				next, err := skipRwset(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthRwset
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
	ErrInvalidLengthRwset = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowRwset   = fmt.Errorf("proto: integer overflow")
)
