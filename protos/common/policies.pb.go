// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: common/policies.proto

package common

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	msp "github.com/rongzer/blockchain/protos/msp"
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

type Policy_PolicyType int32

const (
	Policy_UNKNOWN       Policy_PolicyType = 0
	Policy_SIGNATURE     Policy_PolicyType = 1
	Policy_MSP           Policy_PolicyType = 2
	Policy_IMPLICIT_META Policy_PolicyType = 3
)

var Policy_PolicyType_name = map[int32]string{
	0: "UNKNOWN",
	1: "SIGNATURE",
	2: "MSP",
	3: "IMPLICIT_META",
}

var Policy_PolicyType_value = map[string]int32{
	"UNKNOWN":       0,
	"SIGNATURE":     1,
	"MSP":           2,
	"IMPLICIT_META": 3,
}

func (x Policy_PolicyType) String() string {
	return proto.EnumName(Policy_PolicyType_name, int32(x))
}

func (Policy_PolicyType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_0d02cf0d453425a3, []int{0, 0}
}

type ImplicitMetaPolicy_Rule int32

const (
	ImplicitMetaPolicy_ANY      ImplicitMetaPolicy_Rule = 0
	ImplicitMetaPolicy_ALL      ImplicitMetaPolicy_Rule = 1
	ImplicitMetaPolicy_MAJORITY ImplicitMetaPolicy_Rule = 2
)

var ImplicitMetaPolicy_Rule_name = map[int32]string{
	0: "ANY",
	1: "ALL",
	2: "MAJORITY",
}

var ImplicitMetaPolicy_Rule_value = map[string]int32{
	"ANY":      0,
	"ALL":      1,
	"MAJORITY": 2,
}

func (x ImplicitMetaPolicy_Rule) String() string {
	return proto.EnumName(ImplicitMetaPolicy_Rule_name, int32(x))
}

func (ImplicitMetaPolicy_Rule) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_0d02cf0d453425a3, []int{3, 0}
}

// Policy expresses a policy which the orderer can evaluate, because there has been some desire expressed to support
// multiple policy engines, this is typed as a oneof for now
type Policy struct {
	Type  int32  `protobuf:"varint,1,opt,name=type,proto3" json:"type,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *Policy) Reset()         { *m = Policy{} }
func (m *Policy) String() string { return proto.CompactTextString(m) }
func (*Policy) ProtoMessage()    {}
func (*Policy) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d02cf0d453425a3, []int{0}
}
func (m *Policy) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Policy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Policy.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Policy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Policy.Merge(m, src)
}
func (m *Policy) XXX_Size() int {
	return m.Size()
}
func (m *Policy) XXX_DiscardUnknown() {
	xxx_messageInfo_Policy.DiscardUnknown(m)
}

var xxx_messageInfo_Policy proto.InternalMessageInfo

func (m *Policy) GetType() int32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *Policy) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

// SignaturePolicyEnvelope wraps a SignaturePolicy and includes a version for future enhancements
type SignaturePolicyEnvelope struct {
	Version    int32               `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	Rule       *SignaturePolicy    `protobuf:"bytes,2,opt,name=rule,proto3" json:"rule,omitempty"`
	Identities []*msp.MSPPrincipal `protobuf:"bytes,3,rep,name=identities,proto3" json:"identities,omitempty"`
}

func (m *SignaturePolicyEnvelope) Reset()         { *m = SignaturePolicyEnvelope{} }
func (m *SignaturePolicyEnvelope) String() string { return proto.CompactTextString(m) }
func (*SignaturePolicyEnvelope) ProtoMessage()    {}
func (*SignaturePolicyEnvelope) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d02cf0d453425a3, []int{1}
}
func (m *SignaturePolicyEnvelope) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SignaturePolicyEnvelope) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SignaturePolicyEnvelope.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SignaturePolicyEnvelope) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignaturePolicyEnvelope.Merge(m, src)
}
func (m *SignaturePolicyEnvelope) XXX_Size() int {
	return m.Size()
}
func (m *SignaturePolicyEnvelope) XXX_DiscardUnknown() {
	xxx_messageInfo_SignaturePolicyEnvelope.DiscardUnknown(m)
}

var xxx_messageInfo_SignaturePolicyEnvelope proto.InternalMessageInfo

func (m *SignaturePolicyEnvelope) GetVersion() int32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *SignaturePolicyEnvelope) GetRule() *SignaturePolicy {
	if m != nil {
		return m.Rule
	}
	return nil
}

func (m *SignaturePolicyEnvelope) GetIdentities() []*msp.MSPPrincipal {
	if m != nil {
		return m.Identities
	}
	return nil
}

// SignaturePolicy is a recursive message structure which defines a featherweight DSL for describing
// policies which are more complicated than 'exactly this signature'.  The NOutOf operator is sufficent
// to express AND as well as OR, as well as of course N out of the following M policies
// SignedBy implies that the signature is from a valid certificate which is signed by the trusted
// authority specified in the bytes.  This will be the certificate itself for a self-signed certificate
// and will be the CA for more traditional certificates
type SignaturePolicy struct {
	// Types that are valid to be assigned to Type:
	//	*SignaturePolicy_SignedBy
	//	*SignaturePolicy_NOutOf_
	Type isSignaturePolicy_Type `protobuf_oneof:"Type"`
}

func (m *SignaturePolicy) Reset()         { *m = SignaturePolicy{} }
func (m *SignaturePolicy) String() string { return proto.CompactTextString(m) }
func (*SignaturePolicy) ProtoMessage()    {}
func (*SignaturePolicy) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d02cf0d453425a3, []int{2}
}
func (m *SignaturePolicy) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SignaturePolicy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SignaturePolicy.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SignaturePolicy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignaturePolicy.Merge(m, src)
}
func (m *SignaturePolicy) XXX_Size() int {
	return m.Size()
}
func (m *SignaturePolicy) XXX_DiscardUnknown() {
	xxx_messageInfo_SignaturePolicy.DiscardUnknown(m)
}

var xxx_messageInfo_SignaturePolicy proto.InternalMessageInfo

type isSignaturePolicy_Type interface {
	isSignaturePolicy_Type()
	MarshalTo([]byte) (int, error)
	Size() int
}

type SignaturePolicy_SignedBy struct {
	SignedBy int32 `protobuf:"varint,1,opt,name=signed_by,json=signedBy,proto3,oneof" json:"signed_by,omitempty"`
}
type SignaturePolicy_NOutOf_ struct {
	NOutOf *SignaturePolicy_NOutOf `protobuf:"bytes,2,opt,name=n_out_of,json=nOutOf,proto3,oneof" json:"n_out_of,omitempty"`
}

func (*SignaturePolicy_SignedBy) isSignaturePolicy_Type() {}
func (*SignaturePolicy_NOutOf_) isSignaturePolicy_Type()  {}

func (m *SignaturePolicy) GetType() isSignaturePolicy_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *SignaturePolicy) GetSignedBy() int32 {
	if x, ok := m.GetType().(*SignaturePolicy_SignedBy); ok {
		return x.SignedBy
	}
	return 0
}

func (m *SignaturePolicy) GetNOutOf() *SignaturePolicy_NOutOf {
	if x, ok := m.GetType().(*SignaturePolicy_NOutOf_); ok {
		return x.NOutOf
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*SignaturePolicy) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*SignaturePolicy_SignedBy)(nil),
		(*SignaturePolicy_NOutOf_)(nil),
	}
}

type SignaturePolicy_NOutOf struct {
	N     int32              `protobuf:"varint,1,opt,name=n,proto3" json:"n,omitempty"`
	Rules []*SignaturePolicy `protobuf:"bytes,2,rep,name=rules,proto3" json:"rules,omitempty"`
}

func (m *SignaturePolicy_NOutOf) Reset()         { *m = SignaturePolicy_NOutOf{} }
func (m *SignaturePolicy_NOutOf) String() string { return proto.CompactTextString(m) }
func (*SignaturePolicy_NOutOf) ProtoMessage()    {}
func (*SignaturePolicy_NOutOf) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d02cf0d453425a3, []int{2, 0}
}
func (m *SignaturePolicy_NOutOf) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SignaturePolicy_NOutOf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SignaturePolicy_NOutOf.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SignaturePolicy_NOutOf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignaturePolicy_NOutOf.Merge(m, src)
}
func (m *SignaturePolicy_NOutOf) XXX_Size() int {
	return m.Size()
}
func (m *SignaturePolicy_NOutOf) XXX_DiscardUnknown() {
	xxx_messageInfo_SignaturePolicy_NOutOf.DiscardUnknown(m)
}

var xxx_messageInfo_SignaturePolicy_NOutOf proto.InternalMessageInfo

func (m *SignaturePolicy_NOutOf) GetN() int32 {
	if m != nil {
		return m.N
	}
	return 0
}

func (m *SignaturePolicy_NOutOf) GetRules() []*SignaturePolicy {
	if m != nil {
		return m.Rules
	}
	return nil
}

// ImplicitMetaPolicy is a policy type which depends on the hierarchical nature of the configuration
// It is implicit because the rule is generate implicitly based on the number of sub policies
// It is meta because it depends only on the result of other policies
// When evaluated, this policy iterates over all immediate child sub-groups, retrieves the policy
// of name sub_policy, evaluates the collection and applies the rule.
// For example, with 4 sub-groups, and a policy name of "foo", ImplicitMetaPolicy retrieves
// each sub-group, retrieves policy "foo" for each subgroup, evaluates it, and, in the case of ANY
// 1 satisfied is sufficient, ALL would require 4 signatures, and MAJORITY would require 3 signatures.
type ImplicitMetaPolicy struct {
	SubPolicy string                  `protobuf:"bytes,1,opt,name=sub_policy,json=subPolicy,proto3" json:"sub_policy,omitempty"`
	Rule      ImplicitMetaPolicy_Rule `protobuf:"varint,2,opt,name=rule,proto3,enum=common.ImplicitMetaPolicy_Rule" json:"rule,omitempty"`
}

func (m *ImplicitMetaPolicy) Reset()         { *m = ImplicitMetaPolicy{} }
func (m *ImplicitMetaPolicy) String() string { return proto.CompactTextString(m) }
func (*ImplicitMetaPolicy) ProtoMessage()    {}
func (*ImplicitMetaPolicy) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d02cf0d453425a3, []int{3}
}
func (m *ImplicitMetaPolicy) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ImplicitMetaPolicy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ImplicitMetaPolicy.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ImplicitMetaPolicy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ImplicitMetaPolicy.Merge(m, src)
}
func (m *ImplicitMetaPolicy) XXX_Size() int {
	return m.Size()
}
func (m *ImplicitMetaPolicy) XXX_DiscardUnknown() {
	xxx_messageInfo_ImplicitMetaPolicy.DiscardUnknown(m)
}

var xxx_messageInfo_ImplicitMetaPolicy proto.InternalMessageInfo

func (m *ImplicitMetaPolicy) GetSubPolicy() string {
	if m != nil {
		return m.SubPolicy
	}
	return ""
}

func (m *ImplicitMetaPolicy) GetRule() ImplicitMetaPolicy_Rule {
	if m != nil {
		return m.Rule
	}
	return ImplicitMetaPolicy_ANY
}

func init() {
	proto.RegisterEnum("common.Policy_PolicyType", Policy_PolicyType_name, Policy_PolicyType_value)
	proto.RegisterEnum("common.ImplicitMetaPolicy_Rule", ImplicitMetaPolicy_Rule_name, ImplicitMetaPolicy_Rule_value)
	proto.RegisterType((*Policy)(nil), "common.Policy")
	proto.RegisterType((*SignaturePolicyEnvelope)(nil), "common.SignaturePolicyEnvelope")
	proto.RegisterType((*SignaturePolicy)(nil), "common.SignaturePolicy")
	proto.RegisterType((*SignaturePolicy_NOutOf)(nil), "common.SignaturePolicy.NOutOf")
	proto.RegisterType((*ImplicitMetaPolicy)(nil), "common.ImplicitMetaPolicy")
}

func init() { proto.RegisterFile("common/policies.proto", fileDescriptor_0d02cf0d453425a3) }

var fileDescriptor_0d02cf0d453425a3 = []byte{
	// 512 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x92, 0xcd, 0x6e, 0xd3, 0x40,
	0x10, 0xc7, 0xbd, 0xf9, 0x70, 0x92, 0x49, 0x0a, 0x66, 0x55, 0x94, 0xa8, 0x12, 0x26, 0xb2, 0x10,
	0x8a, 0x54, 0x61, 0x4b, 0x29, 0x27, 0x6e, 0x09, 0x44, 0xd4, 0x10, 0x3b, 0x91, 0x93, 0x0a, 0x95,
	0x8b, 0x15, 0xbb, 0xdb, 0x74, 0x55, 0x7b, 0xd7, 0xf2, 0x47, 0x44, 0x78, 0x8a, 0x9e, 0x78, 0x19,
	0x5e, 0x80, 0x63, 0x8f, 0x1c, 0x51, 0xf2, 0x22, 0xc8, 0xde, 0x04, 0xaa, 0x56, 0xbd, 0xcd, 0x8c,
	0x7f, 0x33, 0xfe, 0xff, 0x67, 0x07, 0x9e, 0xfb, 0x3c, 0x0c, 0x39, 0x33, 0x22, 0x1e, 0x50, 0x9f,
	0x92, 0x44, 0x8f, 0x62, 0x9e, 0x72, 0x2c, 0x8b, 0xf2, 0x51, 0x3b, 0x4c, 0x22, 0x23, 0x4c, 0x22,
	0x37, 0x8a, 0x29, 0xf3, 0x69, 0xb4, 0x08, 0x04, 0xa0, 0x7d, 0x03, 0x79, 0x9a, 0xb7, 0xac, 0x31,
	0x86, 0x4a, 0xba, 0x8e, 0x48, 0x07, 0x75, 0x51, 0xaf, 0xea, 0x14, 0x31, 0x3e, 0x84, 0xea, 0x6a,
	0x11, 0x64, 0xa4, 0x53, 0xea, 0xa2, 0x5e, 0xcb, 0x11, 0x89, 0xf6, 0x01, 0x40, 0xf4, 0xcc, 0x73,
	0xa6, 0x09, 0xb5, 0x33, 0xfb, 0xb3, 0x3d, 0xf9, 0x62, 0x2b, 0x12, 0x3e, 0x80, 0xc6, 0xcc, 0xfc,
	0x68, 0x0f, 0xe6, 0x67, 0xce, 0x48, 0x41, 0xb8, 0x06, 0x65, 0x6b, 0x36, 0x55, 0x4a, 0xf8, 0x19,
	0x1c, 0x98, 0xd6, 0x74, 0x6c, 0xbe, 0x37, 0xe7, 0xae, 0x35, 0x9a, 0x0f, 0x94, 0xb2, 0xf6, 0x03,
	0x41, 0x7b, 0x46, 0x97, 0x6c, 0x91, 0x66, 0x31, 0x11, 0xf3, 0x46, 0x6c, 0x45, 0x02, 0x1e, 0x11,
	0xdc, 0x81, 0xda, 0x8a, 0xc4, 0x09, 0xe5, 0x6c, 0x27, 0x67, 0x9f, 0xe2, 0x63, 0xa8, 0xc4, 0x59,
	0x20, 0x04, 0x35, 0xfb, 0x6d, 0x5d, 0xf8, 0xd3, 0xef, 0x0d, 0x72, 0x0a, 0x08, 0xbf, 0x05, 0xa0,
	0x17, 0x84, 0xa5, 0x34, 0xa5, 0x24, 0xe9, 0x94, 0xbb, 0xe5, 0x5e, 0xb3, 0x7f, 0xb8, 0x6f, 0xb1,
	0x66, 0xd3, 0xe9, 0x7e, 0x19, 0xce, 0x1d, 0x4e, 0xfb, 0x89, 0xe0, 0xe9, 0xbd, 0x79, 0xf8, 0x05,
	0x34, 0x12, 0xba, 0x64, 0xe4, 0xc2, 0xf5, 0xd6, 0x42, 0xd2, 0xa9, 0xe4, 0xd4, 0x45, 0x69, 0xb8,
	0xc6, 0xef, 0xa0, 0xce, 0x5c, 0x9e, 0xa5, 0x2e, 0xbf, 0xdc, 0x29, 0x53, 0x1f, 0x51, 0xa6, 0xdb,
	0x93, 0x2c, 0x9d, 0x5c, 0x9e, 0x4a, 0x8e, 0xcc, 0x8a, 0xe8, 0x68, 0x04, 0xb2, 0xa8, 0xe1, 0x16,
	0xa0, 0xbd, 0x5f, 0xc4, 0xf0, 0x1b, 0xa8, 0xe6, 0x26, 0x92, 0x4e, 0xa9, 0xd0, 0xfd, 0xa8, 0x55,
	0x41, 0x0d, 0x65, 0xa8, 0xe4, 0xcf, 0xa1, 0xdd, 0x20, 0xc0, 0x66, 0x18, 0xe5, 0x57, 0x90, 0x5a,
	0x24, 0x5d, 0xfc, 0x33, 0x00, 0x49, 0xe6, 0xb9, 0xc5, 0x79, 0x08, 0x07, 0x0d, 0xa7, 0x91, 0x64,
	0xde, 0xee, 0xf3, 0xc9, 0x9d, 0xb5, 0x3e, 0xe9, 0xbf, 0xdc, 0xff, 0xeb, 0xe1, 0x20, 0xdd, 0xc9,
	0x02, 0x22, 0xd6, 0xab, 0xbd, 0x86, 0x4a, 0x9e, 0xe5, 0xaf, 0x3c, 0xb0, 0xcf, 0x15, 0xa9, 0x08,
	0xc6, 0x63, 0x05, 0xe1, 0x16, 0xd4, 0xad, 0xc1, 0xa7, 0x89, 0x63, 0xce, 0xcf, 0x95, 0xd2, 0xf0,
	0xfa, 0xd7, 0x46, 0x45, 0xb7, 0x1b, 0x15, 0xfd, 0xd9, 0xa8, 0xe8, 0x66, 0xab, 0x4a, 0xb7, 0x5b,
	0x55, 0xfa, 0xbd, 0x55, 0x25, 0x78, 0xe5, 0xf3, 0x50, 0x8f, 0x39, 0x5b, 0x7e, 0x27, 0xb1, 0xee,
	0x05, 0xdc, 0xbf, 0xf6, 0xaf, 0x16, 0x94, 0x89, 0xdb, 0x4c, 0x76, 0x2a, 0xbe, 0x1e, 0x2f, 0x69,
	0x7a, 0x95, 0x79, 0x79, 0x6a, 0xec, 0x60, 0xe3, 0x3f, 0x6c, 0x08, 0xd8, 0x10, 0xb0, 0x27, 0x17,
	0xe9, 0xc9, 0xdf, 0x00, 0x00, 0x00, 0xff, 0xff, 0x64, 0x56, 0xab, 0x84, 0x11, 0x03, 0x00, 0x00,
}

func (m *Policy) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Policy) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Policy) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(dAtA[i:], m.Value)
		i = encodeVarintPolicies(dAtA, i, uint64(len(m.Value)))
		i--
		dAtA[i] = 0x12
	}
	if m.Type != 0 {
		i = encodeVarintPolicies(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *SignaturePolicyEnvelope) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SignaturePolicyEnvelope) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SignaturePolicyEnvelope) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Identities) > 0 {
		for iNdEx := len(m.Identities) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Identities[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintPolicies(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1a
		}
	}
	if m.Rule != nil {
		{
			size, err := m.Rule.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintPolicies(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if m.Version != 0 {
		i = encodeVarintPolicies(dAtA, i, uint64(m.Version))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *SignaturePolicy) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SignaturePolicy) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SignaturePolicy) MarshalToSizedBuffer(dAtA []byte) (int, error) {
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

func (m *SignaturePolicy_SignedBy) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SignaturePolicy_SignedBy) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	i = encodeVarintPolicies(dAtA, i, uint64(m.SignedBy))
	i--
	dAtA[i] = 0x8
	return len(dAtA) - i, nil
}
func (m *SignaturePolicy_NOutOf_) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SignaturePolicy_NOutOf_) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.NOutOf != nil {
		{
			size, err := m.NOutOf.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintPolicies(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	return len(dAtA) - i, nil
}
func (m *SignaturePolicy_NOutOf) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SignaturePolicy_NOutOf) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SignaturePolicy_NOutOf) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Rules) > 0 {
		for iNdEx := len(m.Rules) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Rules[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintPolicies(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if m.N != 0 {
		i = encodeVarintPolicies(dAtA, i, uint64(m.N))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *ImplicitMetaPolicy) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ImplicitMetaPolicy) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ImplicitMetaPolicy) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Rule != 0 {
		i = encodeVarintPolicies(dAtA, i, uint64(m.Rule))
		i--
		dAtA[i] = 0x10
	}
	if len(m.SubPolicy) > 0 {
		i -= len(m.SubPolicy)
		copy(dAtA[i:], m.SubPolicy)
		i = encodeVarintPolicies(dAtA, i, uint64(len(m.SubPolicy)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintPolicies(dAtA []byte, offset int, v uint64) int {
	offset -= sovPolicies(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Policy) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Type != 0 {
		n += 1 + sovPolicies(uint64(m.Type))
	}
	l = len(m.Value)
	if l > 0 {
		n += 1 + l + sovPolicies(uint64(l))
	}
	return n
}

func (m *SignaturePolicyEnvelope) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Version != 0 {
		n += 1 + sovPolicies(uint64(m.Version))
	}
	if m.Rule != nil {
		l = m.Rule.Size()
		n += 1 + l + sovPolicies(uint64(l))
	}
	if len(m.Identities) > 0 {
		for _, e := range m.Identities {
			l = e.Size()
			n += 1 + l + sovPolicies(uint64(l))
		}
	}
	return n
}

func (m *SignaturePolicy) Size() (n int) {
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

func (m *SignaturePolicy_SignedBy) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	n += 1 + sovPolicies(uint64(m.SignedBy))
	return n
}
func (m *SignaturePolicy_NOutOf_) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.NOutOf != nil {
		l = m.NOutOf.Size()
		n += 1 + l + sovPolicies(uint64(l))
	}
	return n
}
func (m *SignaturePolicy_NOutOf) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.N != 0 {
		n += 1 + sovPolicies(uint64(m.N))
	}
	if len(m.Rules) > 0 {
		for _, e := range m.Rules {
			l = e.Size()
			n += 1 + l + sovPolicies(uint64(l))
		}
	}
	return n
}

func (m *ImplicitMetaPolicy) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.SubPolicy)
	if l > 0 {
		n += 1 + l + sovPolicies(uint64(l))
	}
	if m.Rule != 0 {
		n += 1 + sovPolicies(uint64(m.Rule))
	}
	return n
}

func sovPolicies(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPolicies(x uint64) (n int) {
	return sovPolicies(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Policy) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPolicies
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
			return fmt.Errorf("proto: Policy: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Policy: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
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
				return ErrInvalidLengthPolicies
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthPolicies
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Value = append(m.Value[:0], dAtA[iNdEx:postIndex]...)
			if m.Value == nil {
				m.Value = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPolicies(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPolicies
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPolicies
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
func (m *SignaturePolicyEnvelope) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPolicies
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
			return fmt.Errorf("proto: SignaturePolicyEnvelope: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SignaturePolicyEnvelope: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Version |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rule", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
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
				return ErrInvalidLengthPolicies
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPolicies
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Rule == nil {
				m.Rule = &SignaturePolicy{}
			}
			if err := m.Rule.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Identities", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
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
				return ErrInvalidLengthPolicies
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPolicies
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Identities = append(m.Identities, &msp.MSPPrincipal{})
			if err := m.Identities[len(m.Identities)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPolicies(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPolicies
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPolicies
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
func (m *SignaturePolicy) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPolicies
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
			return fmt.Errorf("proto: SignaturePolicy: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SignaturePolicy: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SignedBy", wireType)
			}
			var v int32
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Type = &SignaturePolicy_SignedBy{v}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NOutOf", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
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
				return ErrInvalidLengthPolicies
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPolicies
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			v := &SignaturePolicy_NOutOf{}
			if err := v.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			m.Type = &SignaturePolicy_NOutOf_{v}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPolicies(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPolicies
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPolicies
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
func (m *SignaturePolicy_NOutOf) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPolicies
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
			return fmt.Errorf("proto: NOutOf: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NOutOf: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field N", wireType)
			}
			m.N = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.N |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rules", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
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
				return ErrInvalidLengthPolicies
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPolicies
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Rules = append(m.Rules, &SignaturePolicy{})
			if err := m.Rules[len(m.Rules)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPolicies(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPolicies
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPolicies
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
func (m *ImplicitMetaPolicy) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPolicies
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
			return fmt.Errorf("proto: ImplicitMetaPolicy: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ImplicitMetaPolicy: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SubPolicy", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
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
				return ErrInvalidLengthPolicies
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPolicies
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SubPolicy = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rule", wireType)
			}
			m.Rule = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicies
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Rule |= ImplicitMetaPolicy_Rule(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipPolicies(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPolicies
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPolicies
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
func skipPolicies(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPolicies
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
					return 0, ErrIntOverflowPolicies
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
					return 0, ErrIntOverflowPolicies
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
				return 0, ErrInvalidLengthPolicies
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthPolicies
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowPolicies
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
				next, err := skipPolicies(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthPolicies
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
	ErrInvalidLengthPolicies = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPolicies   = fmt.Errorf("proto: integer overflow")
)
