// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: msp/msp_principal.proto

package msp

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

type MSPPrincipal_Classification int32

const (
	MSPPrincipal_ROLE MSPPrincipal_Classification = 0
	// one of a member of MSP network, and the one of an
	// administrator of an MSP network
	MSPPrincipal_ORGANIZATION_UNIT MSPPrincipal_Classification = 1
	// groupping of entities, per MSP affiliation
	// E.g., this can well be represented by an MSP's
	// Organization unit
	MSPPrincipal_IDENTITY MSPPrincipal_Classification = 2
)

var MSPPrincipal_Classification_name = map[int32]string{
	0: "ROLE",
	1: "ORGANIZATION_UNIT",
	2: "IDENTITY",
}

var MSPPrincipal_Classification_value = map[string]int32{
	"ROLE":              0,
	"ORGANIZATION_UNIT": 1,
	"IDENTITY":          2,
}

func (x MSPPrincipal_Classification) String() string {
	return proto.EnumName(MSPPrincipal_Classification_name, int32(x))
}

func (MSPPrincipal_Classification) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_82e08b7ead29bd48, []int{0, 0}
}

type MSPRole_MSPRoleType int32

const (
	MSPRole_MEMBER MSPRole_MSPRoleType = 0
	MSPRole_ADMIN  MSPRole_MSPRoleType = 1
)

var MSPRole_MSPRoleType_name = map[int32]string{
	0: "MEMBER",
	1: "ADMIN",
}

var MSPRole_MSPRoleType_value = map[string]int32{
	"MEMBER": 0,
	"ADMIN":  1,
}

func (x MSPRole_MSPRoleType) String() string {
	return proto.EnumName(MSPRole_MSPRoleType_name, int32(x))
}

func (MSPRole_MSPRoleType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_82e08b7ead29bd48, []int{2, 0}
}

// MSPPrincipal aims to represent an MSP-centric set of identities.
// In particular, this structure allows for definition of
//  - a group of identities that are member of the same MSP
//  - a group of identities that are member of the same organization unit
//    in the same MSP
//  - a group of identities that are administering a specific MSP
//  - a specific identity
// Expressing these groups is done given two fields of the fields below
//  - Classification, that defines the type of classification of identities
//    in an MSP this principal would be defined on; Classification can take
//    three values:
//     (i)  ByMSPRole: that represents a classification of identities within
//          MSP based on one of the two pre-defined MSP rules, "member" and "admin"
//     (ii) ByOrganizationUnit: that represents a classification of identities
//          within MSP based on the organization unit an identity belongs to
//     (iii)ByIdentity that denotes that MSPPrincipal is mapped to a single
//          identity/certificate; this would mean that the Principal bytes
//          message
type MSPPrincipal struct {
	// Classification describes the way that one should process
	// Principal. An Classification value of "ByOrganizationUnit" reflects
	// that "Principal" contains the name of an organization this MSP
	// handles. A Classification value "ByIdentity" means that
	// "Principal" contains a specific identity. Default value
	// denotes that Principal contains one of the groups by
	// default supported by all MSPs ("admin" or "member").
	PrincipalClassification MSPPrincipal_Classification `protobuf:"varint,1,opt,name=principal_classification,json=principalClassification,proto3,enum=common.MSPPrincipal_Classification" json:"principal_classification,omitempty"`
	// Principal completes the policy principal definition. For the default
	// principal types, Principal can be either "Admin" or "Member".
	// For the ByOrganizationUnit/ByIdentity values of Classification,
	// PolicyPrincipal acquires its value from an organization unit or
	// identity, respectively.
	Principal []byte `protobuf:"bytes,2,opt,name=principal,proto3" json:"principal,omitempty"`
}

func (m *MSPPrincipal) Reset()         { *m = MSPPrincipal{} }
func (m *MSPPrincipal) String() string { return proto.CompactTextString(m) }
func (*MSPPrincipal) ProtoMessage()    {}
func (*MSPPrincipal) Descriptor() ([]byte, []int) {
	return fileDescriptor_82e08b7ead29bd48, []int{0}
}
func (m *MSPPrincipal) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MSPPrincipal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MSPPrincipal.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MSPPrincipal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MSPPrincipal.Merge(m, src)
}
func (m *MSPPrincipal) XXX_Size() int {
	return m.Size()
}
func (m *MSPPrincipal) XXX_DiscardUnknown() {
	xxx_messageInfo_MSPPrincipal.DiscardUnknown(m)
}

var xxx_messageInfo_MSPPrincipal proto.InternalMessageInfo

func (m *MSPPrincipal) GetPrincipalClassification() MSPPrincipal_Classification {
	if m != nil {
		return m.PrincipalClassification
	}
	return MSPPrincipal_ROLE
}

func (m *MSPPrincipal) GetPrincipal() []byte {
	if m != nil {
		return m.Principal
	}
	return nil
}

// OrganizationUnit governs the organization of the Principal
// field of a policy principal when a specific organization unity members
// are to be defined within a policy principal.
type OrganizationUnit struct {
	// MSPIdentifier represents the identifier of the MSP this organization unit
	// refers to
	MspIdentifier string `protobuf:"bytes,1,opt,name=msp_identifier,json=mspIdentifier,proto3" json:"msp_identifier,omitempty"`
	// OrganizationUnitIdentifier defines the organizational unit under the
	// MSP identified with MSPIdentifier
	OrganizationalUnitIdentifier string `protobuf:"bytes,2,opt,name=organizational_unit_identifier,json=organizationalUnitIdentifier,proto3" json:"organizational_unit_identifier,omitempty"`
	// CertifiersIdentifier is the hash of certificates chain of trust
	// related to this organizational unit
	CertifiersIdentifier []byte `protobuf:"bytes,3,opt,name=certifiers_identifier,json=certifiersIdentifier,proto3" json:"certifiers_identifier,omitempty"`
}

func (m *OrganizationUnit) Reset()         { *m = OrganizationUnit{} }
func (m *OrganizationUnit) String() string { return proto.CompactTextString(m) }
func (*OrganizationUnit) ProtoMessage()    {}
func (*OrganizationUnit) Descriptor() ([]byte, []int) {
	return fileDescriptor_82e08b7ead29bd48, []int{1}
}
func (m *OrganizationUnit) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *OrganizationUnit) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_OrganizationUnit.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *OrganizationUnit) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrganizationUnit.Merge(m, src)
}
func (m *OrganizationUnit) XXX_Size() int {
	return m.Size()
}
func (m *OrganizationUnit) XXX_DiscardUnknown() {
	xxx_messageInfo_OrganizationUnit.DiscardUnknown(m)
}

var xxx_messageInfo_OrganizationUnit proto.InternalMessageInfo

func (m *OrganizationUnit) GetMspIdentifier() string {
	if m != nil {
		return m.MspIdentifier
	}
	return ""
}

func (m *OrganizationUnit) GetOrganizationalUnitIdentifier() string {
	if m != nil {
		return m.OrganizationalUnitIdentifier
	}
	return ""
}

func (m *OrganizationUnit) GetCertifiersIdentifier() []byte {
	if m != nil {
		return m.CertifiersIdentifier
	}
	return nil
}

// MSPRole governs the organization of the Principal
// field of an MSPPrincipal when it aims to define one of the
// two dedicated roles within an MSP: Admin and Members.
type MSPRole struct {
	// MSPIdentifier represents the identifier of the MSP this principal
	// refers to
	MspIdentifier string `protobuf:"bytes,1,opt,name=msp_identifier,json=mspIdentifier,proto3" json:"msp_identifier,omitempty"`
	// MSPRoleType defines which of the available, pre-defined MSP-roles
	// an identiy should posess inside the MSP with identifier MSPidentifier
	Role MSPRole_MSPRoleType `protobuf:"varint,2,opt,name=role,proto3,enum=common.MSPRole_MSPRoleType" json:"role,omitempty"`
}

func (m *MSPRole) Reset()         { *m = MSPRole{} }
func (m *MSPRole) String() string { return proto.CompactTextString(m) }
func (*MSPRole) ProtoMessage()    {}
func (*MSPRole) Descriptor() ([]byte, []int) {
	return fileDescriptor_82e08b7ead29bd48, []int{2}
}
func (m *MSPRole) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MSPRole) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MSPRole.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MSPRole) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MSPRole.Merge(m, src)
}
func (m *MSPRole) XXX_Size() int {
	return m.Size()
}
func (m *MSPRole) XXX_DiscardUnknown() {
	xxx_messageInfo_MSPRole.DiscardUnknown(m)
}

var xxx_messageInfo_MSPRole proto.InternalMessageInfo

func (m *MSPRole) GetMspIdentifier() string {
	if m != nil {
		return m.MspIdentifier
	}
	return ""
}

func (m *MSPRole) GetRole() MSPRole_MSPRoleType {
	if m != nil {
		return m.Role
	}
	return MSPRole_MEMBER
}

func init() {
	proto.RegisterEnum("common.MSPPrincipal_Classification", MSPPrincipal_Classification_name, MSPPrincipal_Classification_value)
	proto.RegisterEnum("common.MSPRole_MSPRoleType", MSPRole_MSPRoleType_name, MSPRole_MSPRoleType_value)
	proto.RegisterType((*MSPPrincipal)(nil), "common.MSPPrincipal")
	proto.RegisterType((*OrganizationUnit)(nil), "common.OrganizationUnit")
	proto.RegisterType((*MSPRole)(nil), "common.MSPRole")
}

func init() { proto.RegisterFile("msp/msp_principal.proto", fileDescriptor_82e08b7ead29bd48) }

var fileDescriptor_82e08b7ead29bd48 = []byte{
	// 416 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0xc1, 0x8e, 0x93, 0x40,
	0x18, 0xc7, 0x99, 0xba, 0xd6, 0xed, 0x67, 0x25, 0x38, 0x71, 0xb3, 0x4d, 0xdc, 0x90, 0x0d, 0xae,
	0x49, 0x4f, 0x90, 0xec, 0x3e, 0x80, 0xe9, 0x5a, 0x62, 0x48, 0x04, 0x9a, 0x59, 0xf6, 0xe0, 0x1e,
	0x24, 0x74, 0x64, 0xdb, 0x89, 0x30, 0x43, 0x06, 0xf6, 0x60, 0x5f, 0xc0, 0xab, 0x0f, 0xe3, 0x43,
	0x78, 0x31, 0xe9, 0xd1, 0xa3, 0x69, 0x5f, 0xc4, 0x00, 0x2d, 0x50, 0x4f, 0x7b, 0x82, 0xf9, 0xfe,
	0xbf, 0xdf, 0x37, 0x33, 0xf0, 0xc1, 0x69, 0x9a, 0x67, 0x56, 0x9a, 0x67, 0x61, 0x26, 0x19, 0xa7,
	0x2c, 0x8b, 0x12, 0x33, 0x93, 0xa2, 0x10, 0xb8, 0x4f, 0x45, 0x9a, 0x0a, 0x6e, 0xfc, 0x46, 0x30,
	0x74, 0x6f, 0x66, 0xb3, 0x7d, 0x8c, 0x3f, 0xc3, 0xa8, 0x61, 0x43, 0x9a, 0x44, 0x79, 0xce, 0xee,
	0x19, 0x8d, 0x0a, 0x26, 0xf8, 0x08, 0x9d, 0xa3, 0xb1, 0x7a, 0xf9, 0xc6, 0xac, 0x5d, 0xb3, 0xeb,
	0x99, 0xef, 0x0f, 0x50, 0x72, 0xda, 0x34, 0x39, 0x0c, 0xf0, 0x19, 0x0c, 0x9a, 0x68, 0xd4, 0x3b,
	0x47, 0xe3, 0x21, 0x69, 0x0b, 0xc6, 0x3b, 0x50, 0xff, 0xe3, 0x8f, 0xe1, 0x88, 0xf8, 0x1f, 0x6d,
	0x4d, 0xc1, 0x27, 0xf0, 0xd2, 0x27, 0x1f, 0x26, 0x9e, 0x73, 0x37, 0x09, 0x1c, 0xdf, 0x0b, 0x6f,
	0x3d, 0x27, 0xd0, 0x10, 0x1e, 0xc2, 0xb1, 0x33, 0xb5, 0xbd, 0xc0, 0x09, 0x3e, 0x69, 0x3d, 0xe3,
	0x27, 0x02, 0xcd, 0x97, 0x8b, 0x88, 0xb3, 0x55, 0xe5, 0xdf, 0x72, 0x56, 0xe0, 0xb7, 0xa0, 0x96,
	0xdf, 0x80, 0x7d, 0x89, 0x79, 0xc1, 0xee, 0x59, 0x2c, 0xab, 0x9b, 0x0c, 0xc8, 0x8b, 0x34, 0xcf,
	0x9c, 0xa6, 0x88, 0xa7, 0xa0, 0x8b, 0x8e, 0x1a, 0x25, 0xe1, 0x03, 0x67, 0x45, 0x57, 0xeb, 0x55,
	0xda, 0xd9, 0x21, 0x55, 0x6e, 0xd1, 0xe9, 0x72, 0x05, 0x27, 0x34, 0x96, 0xf5, 0x22, 0xef, 0xca,
	0x4f, 0xaa, 0xcb, 0xbe, 0x6a, 0xc3, 0x56, 0x32, 0xbe, 0x23, 0x78, 0xe6, 0xde, 0xcc, 0x88, 0x48,
	0xe2, 0xc7, 0x9e, 0xd6, 0x82, 0x23, 0x29, 0x92, 0xb8, 0x3a, 0x93, 0x7a, 0xf9, 0xba, 0xf3, 0x53,
	0xca, 0x2e, 0xfb, 0x67, 0xf0, 0x2d, 0x8b, 0x49, 0x05, 0x1a, 0x17, 0xf0, 0xbc, 0x53, 0xc4, 0x00,
	0x7d, 0xd7, 0x76, 0xaf, 0x6d, 0xa2, 0x29, 0x78, 0x00, 0x4f, 0x27, 0x53, 0xd7, 0xf1, 0x34, 0x74,
	0xbd, 0xfc, 0xb5, 0xd1, 0xd1, 0x7a, 0xa3, 0xa3, 0xbf, 0x1b, 0x1d, 0xfd, 0xd8, 0xea, 0xca, 0x7a,
	0xab, 0x2b, 0x7f, 0xb6, 0xba, 0x02, 0x17, 0x54, 0xa4, 0xa6, 0x14, 0x7c, 0xb1, 0x8a, 0xa5, 0x39,
	0x4f, 0x04, 0xfd, 0x4a, 0x97, 0x11, 0xe3, 0xf5, 0x40, 0xe5, 0xbb, 0xfd, 0xef, 0xc6, 0x0b, 0x56,
	0x2c, 0x1f, 0xe6, 0xe5, 0xd2, 0xda, 0xc1, 0x56, 0x0b, 0x5b, 0x35, 0x5c, 0x8e, 0xe4, 0xbc, 0x5f,
	0xbd, 0x5f, 0xfd, 0x0b, 0x00, 0x00, 0xff, 0xff, 0xb9, 0x9d, 0xfd, 0x16, 0xa4, 0x02, 0x00, 0x00,
}

func (m *MSPPrincipal) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MSPPrincipal) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MSPPrincipal) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Principal) > 0 {
		i -= len(m.Principal)
		copy(dAtA[i:], m.Principal)
		i = encodeVarintMspPrincipal(dAtA, i, uint64(len(m.Principal)))
		i--
		dAtA[i] = 0x12
	}
	if m.PrincipalClassification != 0 {
		i = encodeVarintMspPrincipal(dAtA, i, uint64(m.PrincipalClassification))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *OrganizationUnit) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *OrganizationUnit) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *OrganizationUnit) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.CertifiersIdentifier) > 0 {
		i -= len(m.CertifiersIdentifier)
		copy(dAtA[i:], m.CertifiersIdentifier)
		i = encodeVarintMspPrincipal(dAtA, i, uint64(len(m.CertifiersIdentifier)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.OrganizationalUnitIdentifier) > 0 {
		i -= len(m.OrganizationalUnitIdentifier)
		copy(dAtA[i:], m.OrganizationalUnitIdentifier)
		i = encodeVarintMspPrincipal(dAtA, i, uint64(len(m.OrganizationalUnitIdentifier)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.MspIdentifier) > 0 {
		i -= len(m.MspIdentifier)
		copy(dAtA[i:], m.MspIdentifier)
		i = encodeVarintMspPrincipal(dAtA, i, uint64(len(m.MspIdentifier)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MSPRole) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MSPRole) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MSPRole) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Role != 0 {
		i = encodeVarintMspPrincipal(dAtA, i, uint64(m.Role))
		i--
		dAtA[i] = 0x10
	}
	if len(m.MspIdentifier) > 0 {
		i -= len(m.MspIdentifier)
		copy(dAtA[i:], m.MspIdentifier)
		i = encodeVarintMspPrincipal(dAtA, i, uint64(len(m.MspIdentifier)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintMspPrincipal(dAtA []byte, offset int, v uint64) int {
	offset -= sovMspPrincipal(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MSPPrincipal) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.PrincipalClassification != 0 {
		n += 1 + sovMspPrincipal(uint64(m.PrincipalClassification))
	}
	l = len(m.Principal)
	if l > 0 {
		n += 1 + l + sovMspPrincipal(uint64(l))
	}
	return n
}

func (m *OrganizationUnit) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.MspIdentifier)
	if l > 0 {
		n += 1 + l + sovMspPrincipal(uint64(l))
	}
	l = len(m.OrganizationalUnitIdentifier)
	if l > 0 {
		n += 1 + l + sovMspPrincipal(uint64(l))
	}
	l = len(m.CertifiersIdentifier)
	if l > 0 {
		n += 1 + l + sovMspPrincipal(uint64(l))
	}
	return n
}

func (m *MSPRole) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.MspIdentifier)
	if l > 0 {
		n += 1 + l + sovMspPrincipal(uint64(l))
	}
	if m.Role != 0 {
		n += 1 + sovMspPrincipal(uint64(m.Role))
	}
	return n
}

func sovMspPrincipal(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMspPrincipal(x uint64) (n int) {
	return sovMspPrincipal(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MSPPrincipal) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMspPrincipal
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
			return fmt.Errorf("proto: MSPPrincipal: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MSPPrincipal: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PrincipalClassification", wireType)
			}
			m.PrincipalClassification = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMspPrincipal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PrincipalClassification |= MSPPrincipal_Classification(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Principal", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMspPrincipal
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
				return ErrInvalidLengthMspPrincipal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMspPrincipal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Principal = append(m.Principal[:0], dAtA[iNdEx:postIndex]...)
			if m.Principal == nil {
				m.Principal = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMspPrincipal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMspPrincipal
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthMspPrincipal
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
func (m *OrganizationUnit) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMspPrincipal
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
			return fmt.Errorf("proto: OrganizationUnit: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: OrganizationUnit: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MspIdentifier", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMspPrincipal
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
				return ErrInvalidLengthMspPrincipal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMspPrincipal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MspIdentifier = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OrganizationalUnitIdentifier", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMspPrincipal
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
				return ErrInvalidLengthMspPrincipal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMspPrincipal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OrganizationalUnitIdentifier = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CertifiersIdentifier", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMspPrincipal
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
				return ErrInvalidLengthMspPrincipal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMspPrincipal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CertifiersIdentifier = append(m.CertifiersIdentifier[:0], dAtA[iNdEx:postIndex]...)
			if m.CertifiersIdentifier == nil {
				m.CertifiersIdentifier = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMspPrincipal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMspPrincipal
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthMspPrincipal
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
func (m *MSPRole) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMspPrincipal
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
			return fmt.Errorf("proto: MSPRole: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MSPRole: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MspIdentifier", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMspPrincipal
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
				return ErrInvalidLengthMspPrincipal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMspPrincipal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MspIdentifier = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Role", wireType)
			}
			m.Role = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMspPrincipal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Role |= MSPRole_MSPRoleType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipMspPrincipal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMspPrincipal
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthMspPrincipal
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
func skipMspPrincipal(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMspPrincipal
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
					return 0, ErrIntOverflowMspPrincipal
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
					return 0, ErrIntOverflowMspPrincipal
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
				return 0, ErrInvalidLengthMspPrincipal
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthMspPrincipal
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowMspPrincipal
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
				next, err := skipMspPrincipal(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthMspPrincipal
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
	ErrInvalidLengthMspPrincipal = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMspPrincipal   = fmt.Errorf("proto: integer overflow")
)
