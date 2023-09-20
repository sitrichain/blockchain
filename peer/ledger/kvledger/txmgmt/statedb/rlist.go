// rlist
package statedb

import (
	"fmt"

	"github.com/rongzer/blockchain/common/log"

	"github.com/gogo/protobuf/proto"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/version"
)

var BUCKET_NUM = 100

type RList struct {
	BucketId    string   `protobuf:"bytes,1,opt,name=bucketId" json:"bucketId,omitempty"`
	RListName   string   `protobuf:"bytes,2,opt,name=rListName" json:"rListName,omitempty"`
	BucketSeqNo int64    `protobuf:"bytes,3,opt,name=bucketSeqno" json:"bucketSeqNo,omitempty"`
	ParentList  *RList   `protobuf:"bytes,4,opt,name=parentList" json:"parentList,omitempty"`
	BucketList  []*RList `protobuf:"bytes,5,opt,name=bucketList" json:"bucketList,omitempty"`
	IdList      []string `protobuf:"bytes,6,opt,name=idList" json:"idList,omitempty"`
	Size        int      `protobuf:"bytes,7,opt,name=size" json:"size,omitempty"`
	isLoad      bool
	needSave    bool
	needDelete  bool
	db          VersionedDB
	ns          string
	putStub     map[string]*VersionedValue
	rootlist    *RList
	mapRList    map[string]*RList
}

type RListData struct {
	BucketId       string            `protobuf:"bytes,1,opt,name=bucketId" json:"bucketId,omitempty"`
	BucketSeqNo    int64             `protobuf:"varint,2,opt,name=bucketSeqno" json:"bucketSeqNo,omitempty"`
	ParentBucketId string            `protobuf:"bytes,3,opt,name=parentList" json:"parentBucketId,omitempty"`
	BucketListData []*BucketListData `protobuf:"bytes,4,opt,name=bucketListData" json:"bucketListData,omitempty"`
	IdList         []string          `protobuf:"bytes,5,opt,name=idList" json:"idList,omitempty"`
	Size           int64             `protobuf:"varint,6,opt,name=size" json:"size,omitempty"`
}

//Reset resets
func (cd *RListData) Reset() { *cd = RListData{} }

//String converts to string
func (cd *RListData) String() string { return proto.CompactTextString(cd) }

//ProtoMessage just exists to make proto happy
func (*RListData) ProtoMessage()               {}
func (*RListData) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type BucketListData struct {
	BucketId string `protobuf:"bytes,1,opt,name=bucketId" json:"bucketId,omitempty"`
	Size     int64  `protobuf:"varint,2,opt,name=size" json:"size,omitempty"`
}

//Reset resets
func (cd *BucketListData) Reset() { *cd = BucketListData{} }

//String converts to string
func (cd *BucketListData) String() string { return proto.CompactTextString(cd) }

//ProtoMessage just exists to make proto happy
func (*BucketListData) ProtoMessage()               {}
func (*BucketListData) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

var fileDescriptor0 = []byte{
	// 900 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x84, 0x55, 0xdf, 0x6e, 0xe3, 0xc4,
	0x1b, 0xad, 0xe3, 0xfc, 0x69, 0xbe, 0x34, 0xad, 0x3b, 0xd9, 0xfe, 0xd6, 0xbf, 0xc2, 0x6a, 0x23,
	0xc3, 0xa2, 0xd2, 0x4a, 0x89, 0x28, 0x37, 0x70, 0xe9, 0xd8, 0x93, 0xd6, 0x6a, 0xd6, 0x2e, 0x63,
	0x67, 0x11, 0xbb, 0x48, 0x96, 0x93, 0x4c, 0x13, 0x8b, 0xc4, 0x8e, 0x6c, 0xa7, 0x6a, 0x5f, 0x02,
	0x21, 0xc1, 0x0d, 0x17, 0xbc, 0x00, 0x4f, 0xc2, 0x5b, 0xf0, 0x12, 0x48, 0xdc, 0xa2, 0xf1, 0x8c,
	0xbd, 0x49, 0x59, 0x89, 0xab, 0xcc, 0x39, 0x73, 0x3c, 0xdf, 0x99, 0xf3, 0x7d, 0xb1, 0xa1, 0x33,
	0x8d, 0x57, 0xab, 0x38, 0xea, 0xf3, 0x9f, 0xde, 0x3a, 0x89, 0xb3, 0x18, 0xd5, 0x39, 0x3a, 0x7d,
	0x39, 0x8f, 0xe3, 0xf9, 0x92, 0xf6, 0x73, 0x76, 0xb2, 0xb9, 0xeb, 0x67, 0xe1, 0x8a, 0xa6, 0x59,
	0xb0, 0x5a, 0x73, 0xa1, 0xa6, 0x01, 0x8c, 0x82, 0x34, 0x33, 0xe2, 0xe8, 0x2e, 0x9c, 0xa3, 0x67,
	0x50, 0x0b, 0xa3, 0x19, 0x7d, 0x50, 0xa5, 0xae, 0x74, 0x56, 0x25, 0x1c, 0x68, 0xef, 0x60, 0xff,
	0x35, 0xcd, 0x82, 0x59, 0x90, 0x05, 0x4c, 0x71, 0x1f, 0x2c, 0x37, 0x34, 0x57, 0x1c, 0x10, 0x0e,
	0xd0, 0xd7, 0x00, 0x69, 0x38, 0x8f, 0x82, 0x6c, 0x93, 0xd0, 0x54, 0xad, 0x74, 0xe5, 0xb3, 0xd6,
	0xe5, 0xff, 0x7b, 0xc2, 0x51, 0xf1, 0xac, 0x5b, 0x28, 0xc8, 0x96, 0x58, 0xfb, 0x1e, 0x8e, 0xff,
	0x25, 0x40, 0x9f, 0x83, 0x52, 0x4a, 0xfc, 0x05, 0x0d, 0x66, 0x34, 0x11, 0x05, 0x8f, 0x4a, 0xfe,
	0x3a, 0xa7, 0xd1, 0xc7, 0xd0, 0x2c, 0x29, 0xb5, 0x92, 0x6b, 0xde, 0x13, 0xda, 0x5b, 0xa8, 0x0b,
	0xdd, 0x2b, 0x38, 0x9c, 0x2e, 0x82, 0x28, 0xa2, 0xcb, 0xdd, 0x03, 0xdb, 0x82, 0x15, 0xb2, 0x0f,
	0x55, 0xae, 0x7c, 0xb0, 0xb2, 0xf6, 0xa7, 0x04, 0x6d, 0x63, 0xe7, 0x61, 0x04, 0xd5, 0xec, 0x71,
	0xcd, 0xb3, 0xa9, 0x91, 0x7c, 0x8d, 0x54, 0x68, 0xdc, 0xd3, 0x24, 0x0d, 0xe3, 0x28, 0x3f, 0xa7,
	0x46, 0x0a, 0x88, 0xbe, 0x82, 0x66, 0xd9, 0x0d, 0x55, 0xee, 0x4a, 0x67, 0xad, 0xcb, 0xd3, 0x1e,
	0xef, 0x57, 0xaf, 0xe8, 0x57, 0xcf, 0x2b, 0x14, 0xe4, 0xbd, 0x18, 0xbd, 0x00, 0x28, 0xee, 0x12,
	0xce, 0xd4, 0x6a, 0x57, 0x3a, 0x6b, 0x92, 0xa6, 0x60, 0xac, 0x19, 0xea, 0x40, 0x2d, 0x7b, 0x60,
	0x3b, 0xb5, 0x7c, 0xa7, 0x9a, 0x3d, 0x58, 0x33, 0xd6, 0x38, 0xba, 0x8e, 0xa7, 0x0b, 0xb5, 0xce,
	0x5b, 0x9b, 0x03, 0x96, 0x1e, 0x7d, 0xc8, 0x68, 0x94, 0xfb, 0x6b, 0xf0, 0xf4, 0x4a, 0x42, 0xd3,
	0xe1, 0xc8, 0x7d, 0x12, 0xb7, 0x0a, 0x8d, 0x69, 0x42, 0x83, 0x2c, 0x2e, 0xf2, 0x2b, 0x20, 0x2b,
	0x10, 0xc5, 0xd1, 0xb4, 0x68, 0x02, 0x07, 0x1a, 0x86, 0xc6, 0x6d, 0xf0, 0xb8, 0x8c, 0x83, 0x19,
	0xfa, 0x0c, 0xea, 0x5b, 0xc9, 0xb7, 0x2e, 0x0f, 0x8b, 0x01, 0xe1, 0x47, 0x13, 0xb1, 0xcb, 0x52,
	0x64, 0xd3, 0x20, 0xce, 0xc9, 0xd7, 0xda, 0x00, 0xf6, 0x71, 0x74, 0x4f, 0x97, 0x31, 0x4f, 0x74,
	0xcd, 0x8f, 0x2c, 0x2c, 0x08, 0xf8, 0x1f, 0xb3, 0xf0, 0xa3, 0x04, 0xb5, 0xc1, 0x32, 0x9e, 0xfe,
	0x80, 0x2e, 0x9e, 0x38, 0xe9, 0x14, 0x4e, 0xf2, 0xed, 0x27, 0x76, 0x5e, 0x6d, 0xd9, 0x69, 0x5d,
	0x1e, 0xef, 0x48, 0xcd, 0x20, 0x0b, 0xb8, 0x43, 0xf4, 0x05, 0xec, 0xaf, 0xc4, 0x1c, 0x8b, 0x66,
	0x9e, 0xec, 0x48, 0x8b, 0x21, 0x27, 0xa5, 0x4c, 0x9b, 0x43, 0x6b, 0xab, 0x20, 0xfa, 0x1f, 0xd4,
	0xa3, 0xcd, 0x6a, 0x22, 0x5c, 0x55, 0x89, 0x40, 0xe8, 0x13, 0x68, 0xaf, 0x13, 0x7a, 0x1f, 0xc6,
	0x9b, 0xd4, 0x5f, 0x04, 0xe9, 0x42, 0xdc, 0xec, 0xa0, 0x20, 0xaf, 0x83, 0x74, 0x81, 0x3e, 0x82,
	0x26, 0x3b, 0x93, 0x0b, 0xe4, 0x5c, 0xb0, 0xcf, 0x08, 0xb6, 0xa9, 0xbd, 0x84, 0x66, 0x69, 0xb7,
	0x8c, 0x57, 0xea, 0xca, 0x65, 0xbc, 0x17, 0xd0, 0xde, 0x31, 0x89, 0x4e, 0xb7, 0x6e, 0xc3, 0x85,
	0x25, 0x3e, 0xff, 0x5d, 0x82, 0xba, 0x9b, 0x05, 0xd9, 0x26, 0x45, 0x2d, 0x68, 0x8c, 0xed, 0x1b,
	0xdb, 0xf9, 0xd6, 0x56, 0xf6, 0xd0, 0x01, 0x34, 0xdc, 0xb1, 0x61, 0x60, 0xd7, 0x55, 0xfe, 0x90,
	0x90, 0x02, 0xad, 0x81, 0x6e, 0xfa, 0x04, 0x7f, 0x33, 0xc6, 0xae, 0xa7, 0xfc, 0x24, 0xa3, 0x43,
	0x68, 0x0e, 0x1d, 0x32, 0xb0, 0x4c, 0x13, 0xdb, 0xca, 0xcf, 0x39, 0xb6, 0x1d, 0xcf, 0x1f, 0x3a,
	0x63, 0xdb, 0x54, 0x7e, 0x91, 0xd1, 0x0b, 0x50, 0x85, 0xda, 0xc7, 0xb6, 0x67, 0x79, 0xdf, 0xf9,
	0x9e, 0xe3, 0xf8, 0x23, 0x9d, 0x5c, 0x61, 0xe5, 0x37, 0x19, 0x9d, 0xc2, 0x89, 0x65, 0x7b, 0x98,
	0xd8, 0xfa, 0xc8, 0x77, 0x31, 0x79, 0x83, 0x89, 0x8f, 0x09, 0x71, 0x88, 0xf2, 0x97, 0x8c, 0x54,
	0xe8, 0x30, 0xca, 0x32, 0xb0, 0x3f, 0xb6, 0xf5, 0x37, 0xba, 0x35, 0xd2, 0x07, 0x23, 0xac, 0xfc,
	0x2d, 0x9f, 0xff, 0x2a, 0x01, 0xf0, 0x7c, 0x3d, 0xf6, 0x6f, 0x6c, 0x41, 0xe3, 0x35, 0x76, 0x5d,
	0xfd, 0x0a, 0x2b, 0x7b, 0x08, 0xa0, 0x6e, 0x38, 0xf6, 0xd0, 0xba, 0x52, 0x24, 0x74, 0x0c, 0x6d,
	0xbe, 0xf6, 0xc7, 0xb7, 0xa6, 0xee, 0x61, 0xa5, 0x82, 0x54, 0x78, 0x86, 0x6d, 0xd3, 0x21, 0x2e,
	0x26, 0xbe, 0x47, 0x74, 0xdb, 0xd5, 0x0d, 0xcf, 0x72, 0x6c, 0x45, 0x46, 0xcf, 0xa1, 0xe3, 0x10,
	0x13, 0x93, 0x27, 0x1b, 0x55, 0x74, 0x02, 0xc7, 0x26, 0x1e, 0x59, 0xcc, 0x9b, 0x8b, 0xf1, 0x8d,
	0x6f, 0xd9, 0x43, 0x47, 0xa9, 0x31, 0xda, 0xb8, 0xd6, 0x2d, 0xdb, 0x70, 0x4c, 0xec, 0xdf, 0xea,
	0xc6, 0x0d, 0xab, 0x5f, 0x3f, 0x7f, 0x07, 0x68, 0x27, 0x75, 0x8b, 0xbd, 0x6d, 0xd1, 0x21, 0x80,
	0x6b, 0x5d, 0xd9, 0xba, 0x37, 0x26, 0xd8, 0x55, 0xf6, 0xd0, 0x11, 0xb4, 0x46, 0xba, 0xeb, 0xf9,
	0xa5, 0xd5, 0xe7, 0xd0, 0xd9, 0xaa, 0xea, 0xfa, 0x43, 0x6b, 0xe4, 0x61, 0xa2, 0x54, 0xd8, 0xe5,
	0x84, 0x2d, 0x45, 0x1e, 0xb8, 0xf0, 0x69, 0x9c, 0xcc, 0x7b, 0x8b, 0xc7, 0x35, 0x4d, 0x96, 0x74,
	0x36, 0xa7, 0x49, 0xef, 0x2e, 0x98, 0x24, 0xe1, 0x94, 0xbf, 0x5b, 0x52, 0x31, 0x9c, 0x6f, 0x2f,
	0xe6, 0x61, 0xb6, 0xd8, 0x4c, 0x18, 0xec, 0x6f, 0x89, 0xfb, 0x5c, 0xcc, 0x3f, 0x1c, 0xa9, 0xf8,
	0xb8, 0x4c, 0xea, 0x39, 0xfc, 0xf2, 0x9f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xa1, 0xcb, 0xe1, 0xb4,
	0x74, 0x06, 0x00, 0x00,
}

func NewRList(db VersionedDB, ns string, rListName string) *RList {
	root := new(RList)
	root.BucketId = "__RLIST_" + rListName + "_ROOT"
	root.RListName = rListName
	root.BucketSeqNo = 0
	root.BucketList = make([]*RList, 0)
	root.IdList = []string{}
	root.ns = ns
	root.db = db
	root.rootlist = root
	root.mapRList = make(map[string]*RList)
	root.putStub = make(map[string]*VersionedValue)
	root.mapRList[root.BucketId] = root
	root.needSave = true

	root.load()

	if len(root.BucketList) < 1 {
		child := root.newBucket(rListName, root)
		child.ParentList = root
		child.isLoad = true
		root.BucketList = append(root.BucketList, child)
	}
	return root
}

func (this *RList) SetRootDB(db VersionedDB) {
	this.rootlist.db = db
}

func (this *RList) newBucket(rListName string, rootlist *RList) *RList {
	child := new(RList)
	seqno := rootlist.BucketSeqNo + 1
	rootlist.BucketSeqNo = seqno
	child.BucketId = "__RLIST_" + rListName + "_BUCKET_" + fmt.Sprint(seqno)
	child.RListName = rListName
	child.BucketSeqNo = seqno
	child.BucketList = make([]*RList, 0)
	child.IdList = []string{}

	child.needSave = true
	child.ns = this.ns
	child.rootlist = rootlist
	rootlist.mapRList[child.BucketId] = child

	return child
}
func (this *RList) newRListByBucketId(rListName, bucketId string, rootlist *RList) *RList {
	rlist := new(RList)

	rlist.BucketId = bucketId
	rlist.RListName = rListName
	rlist.BucketList = make([]*RList, 0)
	rlist.IdList = []string{}
	rlist.ns = this.ns
	rlist.rootlist = rootlist
	rootlist.mapRList[rlist.BucketId] = rlist

	return rlist
}

func (this *RList) Print(flag int) {
	this.printEx(flag, "")
}

func (this *RList) printEx(flag int, indent string) {
	if flag == 0 {
		this.load()
	}

	if len(this.BucketList) > 0 {
		log.Logger.Debug(indent+"BucketSeqNo:", this.BucketSeqNo, " Size:", this.Size, " BucketId:", this.BucketId, " isLoad:", fmt.Sprint(this.isLoad), " needSave:", fmt.Sprint(this.needSave), " needDelete:", fmt.Sprint(this.needDelete))
		if this.ParentList != nil {
			log.Logger.Debug(" ParentBucketId:", this.ParentList.BucketId)
		}
		log.Logger.Debug(" BucketList [")
		for _, v := range this.BucketList {
			log.Logger.Debug(v.BucketId + " ")
		}
		log.Logger.Debug("]\n")

		for _, v := range this.BucketList {
			v.printEx(flag, indent+"  ")
		}
	} else {
		log.Logger.Debug(indent+"BucketSeqNo:", this.BucketSeqNo, " Size:", this.Size, " BucketId:", this.BucketId, " isLoad:", fmt.Sprint(this.isLoad), " needSave:", fmt.Sprint(this.needSave), " needDelete:", fmt.Sprint(this.needDelete))
		if this.ParentList != nil {
			log.Logger.Debug(" ParentBucketId:", this.ParentList.BucketId)
		}
		log.Logger.Debug(" IdList", this.IdList, "\n")
	}
}

func (this *RList) PrintAll(indent string) string {
	this.load()
	sreturn := ""

	if len(this.BucketList) > 0 {
		sreturn += fmt.Sprint(indent+"BucketSeqNo:", this.BucketSeqNo, " Size:", this.Size, " BucketId:", this.BucketId, " isLoad:", fmt.Sprint(this.isLoad), " needSave:", fmt.Sprint(this.needSave), " needDelete:", fmt.Sprint(this.needDelete))
		if this.ParentList != nil {
			sreturn += fmt.Sprint(" ParentBucketId:", this.ParentList.BucketId)
		}
		sreturn += fmt.Sprint(" BucketList [")
		for _, v := range this.BucketList {
			sreturn += fmt.Sprint(v.BucketId + " ")
		}
		sreturn += fmt.Sprint("]")

		for _, v := range this.BucketList {
			sreturn += v.PrintAll(indent + "  ")
		}
	} else {
		sreturn += fmt.Sprint(indent+"BucketSeqNo:", this.BucketSeqNo, " Size:", this.Size, " BucketId:", this.BucketId, " isLoad:", fmt.Sprint(this.isLoad), " needSave:", fmt.Sprint(this.needSave), " needDelete:", fmt.Sprint(this.needDelete))
		if this.ParentList != nil {
			sreturn += fmt.Sprint(" ParentBucketId:", this.ParentList.BucketId)
		}
		sreturn += fmt.Sprint(" IdList", this.IdList)
	}
	return sreturn
}

func (this *RList) load() *RList {
	if !this.isLoad {
		versionValue, _ := this.rootlist.db.GetState(this.rootlist.ns, this.BucketId)
		if versionValue == nil || len(versionValue.Value) == 0 {
			this.isLoad = true
			return this
		}

		rdata := &RListData{}
		_ = proto.Unmarshal(versionValue.Value, rdata)

		this.BucketId = rdata.BucketId
		this.BucketSeqNo = rdata.BucketSeqNo
		this.Size = int(rdata.Size)
		this.IdList = rdata.IdList

		if rdata.ParentBucketId != "" {
			parent, _ := this.rootlist.mapRList[rdata.ParentBucketId]
			if parent == nil {
				parent = this.newRListByBucketId(this.RListName, rdata.ParentBucketId, this.rootlist)
			}
			parent.load()
			this.ParentList = parent

		}

		bucketlist := make([]*RList, 0)
		for _, v := range rdata.BucketListData {
			bucket := this.rootlist.mapRList[v.BucketId]
			if bucket == nil {
				bucket = new(RList)
				bucket.BucketId = v.BucketId
				bucket.RListName = this.RListName
				bucket.ParentList = this
				bucket.Size = int(v.Size)
				bucket.ns = this.ns
				bucket.rootlist = this.rootlist
				bucket.isLoad = false
				bucket.needSave = false

				this.rootlist.mapRList[v.BucketId] = bucket
			}

			bucketlist = append(bucketlist, bucket)
		}
		this.BucketList = bucketlist

		this.isLoad = true
		this.needSave = false
		this.rootlist.mapRList[this.BucketId] = this
	}

	return this
}
func (this *RList) addSize(num int) {
	this.Size = this.Size + num
	if this.ParentList != nil {
		this.ParentList.addSize(num)
	}
}
func (this *RList) setSize() {
	if len(this.BucketList) > 0 {
		var calSize = 0
		for _, v := range this.BucketList {
			calSize += v.Size
		}
		this.Size = calSize
	}
	if this.ParentList != nil {
		this.ParentList.setSize()
	}
}
func (this *RList) setNeedSave() {
	this.needSave = true
	if this.ParentList != nil {
		this.ParentList.setNeedSave()
	}
}

func (this *RList) AddId(vh *version.Height, id string) {
	this.AddIndexId(vh, this.Size, id)
}

func (this *RList) AddIndexId(vh *version.Height, index int, id string) {
	this.addIndexId(vh, index, id, true)
}

func (this *RList) addIndexId(vh *version.Height, index int, id string, needRemove bool) {
	if this.rootlist != nil && needRemove {
		if this.BucketId == this.rootlist.BucketId {
			idIndex := this.IndexOf(id)
			if idIndex >= 0 {
				this.RemoveIndex(idIndex)
			}
		}
	}

	if index < 0 {
		index = 0
	}
	if index > this.Size {
		index = this.Size
	}
	this.load()
	if len(this.BucketList) > 0 {
		if index >= this.Size {
			bucket := this.BucketList[len(this.BucketList)-1]
			bucket.AddId(vh, id)
		} else {
			for _, v := range this.BucketList {
				if v.Size > index {
					v.addIndexId(vh, index, id, false)
					break
				} else {
					index = index - v.Size
				}
			}
		}

		if len(this.BucketList) > BUCKET_NUM*15/10 && this.ParentList != nil {
			curIndex := indexOf(this.ParentList.BucketList, this)
			theList := this.BucketList
			//bucketlist := make([]*RList, 0)

			next := this.newBucket(this.RListName, this.rootlist)
			next.isLoad = true
			next.BucketList = theList[BUCKET_NUM:]

			for _, v := range next.BucketList {
				v.load()
				v.ParentList = next
				v.needSave = true
			}
			next.setSize()

			this.BucketList = theList[:BUCKET_NUM]
			this.setSize()
			this.ParentList.addBucket(curIndex+1, next)
			this.ParentList.setSize()
		} else if len(this.BucketList) > BUCKET_NUM*15/10 {
			theList := this.BucketList
			bucketlist := make([]*RList, 0)
			first := this.newBucket(this.RListName, this.rootlist)
			first.isLoad = true
			first.BucketList = theList[:BUCKET_NUM]
			for _, v := range first.BucketList {
				v.load()
				v.ParentList = first
				v.needSave = true
			}
			first.setSize()

			next := this.newBucket(this.RListName, this.rootlist)
			next.isLoad = true
			next.BucketList = theList[BUCKET_NUM:]
			for _, v := range next.BucketList {
				v.load()
				v.ParentList = next
				v.needSave = true
			}
			next.setSize()
			this.BucketList = bucketlist
			this.addBucket(0, first)
			this.addBucket(1, next)
			this.setSize()
		}

	} else {
		var idlist []string
		if index >= this.Size {
			idlist = append(this.IdList, id)
		} else {
			idlist = append(idlist, this.IdList[:index]...)
			idlist = append(idlist, id)
			idlist = append(idlist, this.IdList[index:]...)
		}
		this.IdList = idlist
		this.addSize(1)
		this.setNeedSave()
		this.rootlist.putStub["__RLIST_"+this.RListName+"_DATAB_"+id] = &VersionedValue{[]byte(this.BucketId), vh}

		if len(this.IdList) >= BUCKET_NUM*15/10 && this.ParentList != nil {
			this.IdList = idlist[:BUCKET_NUM]
			this.Size = len(this.IdList)
			this.needSave = true
			if index < BUCKET_NUM {
				this.rootlist.putStub["__RLIST_"+this.RListName+"_DATAB_"+id] = &VersionedValue{[]byte(this.BucketId), vh}
			}

			next := this.newBucket(this.RListName, this.rootlist)
			next.ParentList = this.ParentList
			next.IdList = idlist[BUCKET_NUM:]
			next.Size = len(next.IdList)
			next.needSave = true

			for _, v := range next.IdList {

				//数据移动时高度不变化
				var vh1 *version.Height
				putValue := this.rootlist.putStub["__RLIST_"+this.RListName+"_DATAB_"+v]
				if putValue == nil {
					putValue, _ = this.rootlist.db.GetState(this.rootlist.ns, "__RLIST_"+this.RListName+"_DATAB_"+v)
				}

				if putValue != nil {
					vh1 = putValue.Version
				} else {
					vh1 = vh
				}

				this.rootlist.putStub["__RLIST_"+next.RListName+"_DATAB_"+v] = &VersionedValue{[]byte(next.BucketId), vh1}
			}

			curBucketIndex := 0
			for k, v := range this.ParentList.BucketList {
				if v.BucketId == this.BucketId {
					curBucketIndex = k
					break
				}
			}
			this.ParentList.addBucket(curBucketIndex+1, next)
		}
	}
}

func (this *RList) addBucket(index int, bucket *RList) {
	bucketlist := make([]*RList, 0)
	if index >= len(this.BucketList) {
		bucketlist = append(this.BucketList, bucket)
	} else {
		bucketlist = append(bucketlist, this.BucketList[:index]...)
		bucketlist = append(bucketlist, bucket)
		bucketlist = append(bucketlist, this.BucketList[index:]...)
	}
	bucket.ParentList = this
	this.BucketList = bucketlist
	bucket.setNeedSave()
}

//获取某个块高度的所有值，以及下个有数据的块高度
func (this *RList) GetIdsByHeight(height int) ([]string, int) {

	begin := 0
	end := this.rootlist.Size - 1
	if end-begin > 100 {
		begin = this.findHeightPos(height, begin, end)
		if begin < 0 {
			begin = 0
		}
	}

	var ids []string
	nextHeight := -1
	// 处理查找height
	for i := begin; i <= end; i++ {
		value := this.GetIdByIndex(i)
		theHeight := this.GetIdHeight(value)
		if theHeight < height {
			continue
		}

		//中止循环
		if theHeight > height {
			nextHeight = theHeight
			break
		}

		if theHeight == height {
			ids = append(ids, value)
		}
	}
	return ids, nextHeight
}

func (this *RList) findHeightPos(height int, begin int, end int) int {
	if begin < 0 {
		begin = 0
	}
	if end < 0 {
		return -1
	}

	if end-begin < 100 {
		return begin
	}

	beginHeight := this.GetIndexHeight(begin)
	endHeight := this.GetIndexHeight(end)

	if beginHeight >= height {
		return begin
	}

	if endHeight < height {
		return end + 1
	}

	middle := (begin + end) / 2
	middleHeight := this.GetIndexHeight(middle)

	if height <= middleHeight {
		return this.findHeightPos(height, begin, middle)
	}
	return this.findHeightPos(height, middle, end)

}

func (this *RList) GetIndexHeight(index int) int {
	id := this.GetIdByIndex(index)
	if len(id) < 1 {
		return -1
	}

	return this.GetIdHeight(id)
}

func (this *RList) GetIdHeight(id string) int {
	idVersionValue, _ := this.rootlist.db.GetState(this.rootlist.ns, "__RLIST_"+this.RListName+"_DATAB_"+id)
	if idVersionValue == nil {
		return -1
	}
	return int(idVersionValue.Version.BlockNum)
}

func (this *RList) GetIdByIndex(index int) string {
	this.load()
	if index >= this.Size || index < 0 {
		return ""
	}
	if len(this.BucketList) > 0 {
		var id string
		for _, v := range this.BucketList {
			if index >= v.Size {
				index = index - v.Size
			} else {
				id = v.GetIdByIndex(index)
				break
			}
		}
		return id
	} else {
		if index >= len(this.IdList) {
			this.Print(0)
			log.Logger.Debugf("GetIdByIndex %d,%d\n", index, len(this.IdList))

			return ""
		}
		return this.IdList[index]
	}
}

func (this *RList) getIndex() int {
	var num = 0
	if this.ParentList != nil {
		num += this.ParentList.getIndex()
		for _, v := range this.ParentList.BucketList {
			if v.BucketId == this.BucketId {
				break
			} else {
				num += v.Size
			}
		}
	}
	return num
}

func (this *RList) IndexOf(id string) int {
	var num = -1
	this.load()
	putValue := this.rootlist.putStub["__RLIST_"+this.RListName+"_DATAB_"+id]
	bucketId := ""
	if putValue == nil {
		versionValue, _ := this.rootlist.db.GetState(this.rootlist.ns, "__RLIST_"+this.RListName+"_DATAB_"+id)
		if versionValue == nil || len(versionValue.Value) == 0 {
			return num
		}
		bucketId = string(versionValue.Value)
	} else {
		bucketId = string(putValue.Value)
	}

	if bucketId != "" {
		rList, _ := this.rootlist.mapRList[bucketId]
		if rList == nil {
			rList = this.newRListByBucketId(this.RListName, bucketId, this.rootlist)
		}
		rList.load()

		num = rList.getIndex()
		for _, vid := range rList.IdList {
			if vid == id {
				break
			} else {
				num++
			}
		}

	}

	return num
}

func (this *RList) RemoveId(id string) {
	index := this.IndexOf(id)
	this.RemoveIndex(index)
}
func (this *RList) RemoveIndex(index int) {
	this.removeIdx(index, index)
}

func (this *RList) removeIdx(index int, absouteIndex int) {
	if index < 0 || index >= this.Size {
		return
	}
	this.load()
	if len(this.BucketList) > 0 {
		if index > this.Size {
			bucket := this.BucketList[len(this.BucketList)-1]
			bucket.removeIdx(index, absouteIndex)
		} else {
			for _, v := range this.BucketList {
				if index >= v.Size {
					index = index - v.Size
				} else {
					v.removeIdx(index, absouteIndex)
					break
				}
			}
		}

		//当子数据块少于一半时，并且兄弟结点超过一个时
		if len(this.BucketList) < BUCKET_NUM*5/10 && this.ParentList != nil && len(this.ParentList.BucketList) > 1 {
			bucketIndex := 0
			for k, v := range this.ParentList.BucketList {
				if v.BucketId == this.BucketId {
					bucketIndex = k
					break
				}
			}

			addBucket := this.ParentList.BucketList[1]
			if bucketIndex > 0 {
				addBucket = this.ParentList.BucketList[bucketIndex-1]
			}

			addBucket.load()

			this.ParentList.addSize(-1 * this.Size)
			addBucket.addSize(this.Size)

			for k, v := range this.BucketList {
				v.load()
				if bucketIndex == 0 {
					addBucket.addBucket(k, v)
				} else {
					addBucket.addBucket(len(addBucket.BucketList), v)
				}
				v.ParentList = addBucket
				v.setNeedSave()
			}

			if bucketIndex == 0 {
				this.ParentList.BucketList = this.ParentList.BucketList[1:]
			} else {
				bucketlist := make([]*RList, 0)
				bucketlist = append(bucketlist, this.ParentList.BucketList[:bucketIndex]...)
				bucketlist = append(bucketlist, this.ParentList.BucketList[bucketIndex+1:]...)
				this.ParentList.BucketList = bucketlist
			}
			this.needDelete = true
		}

		//去除此Bucket
		if this.Size == 0 && this.ParentList != nil {
			bucketIndex := 0
			this.setNeedSave()
			for k, v := range this.ParentList.BucketList {
				if v.BucketId == this.BucketId {
					bucketIndex = k
					break
				}
			}

			if bucketIndex == 0 {
				this.ParentList.BucketList = this.ParentList.BucketList[1:]
			} else {
				bucketlist := make([]*RList, 0)
				bucketlist = append(bucketlist, this.ParentList.BucketList[:bucketIndex]...)
				bucketlist = append(bucketlist, this.ParentList.BucketList[bucketIndex+1:]...)
				this.ParentList.BucketList = bucketlist
			}
			this.needDelete = true
		}
	} else {
		if index < 0 || index >= len(this.IdList) {
			log.Logger.Debug("=========================================================================================\n")
			this.rootlist.Print(0)
			log.Logger.Debug("=========================================================================================\n")
			this.Print(0)
			log.Logger.Debug("=========================================================================================\n")

			log.Logger.Debugf("remove idx %d,%d\n", index, absouteIndex)
		}
		this.addSize(-1)
		this.setNeedSave()

		id := this.IdList[index]
		this.rootlist.putStub["__RLIST_"+this.RListName+"_DATAB_"+id] = nil

		if index == 0 {
			this.IdList = this.IdList[1:]
		} else {
			var idlist []string
			idlist = append(idlist, this.IdList[:index]...)
			idlist = append(idlist, this.IdList[index+1:]...)
			this.IdList = idlist
		}

		if len(this.IdList) < BUCKET_NUM*5/10 && this.rootlist.Size > BUCKET_NUM*5 {
			//idlist数量小于0.5倍，有父级节点且有兄弟节点
			this.addSize(-1 * len(this.IdList))
			this.needDelete = true

			bucketIndex := 0
			for k, v := range this.ParentList.BucketList {
				if v.BucketId == this.BucketId {
					bucketIndex = k
					break
				}
			}
			if bucketIndex == 0 {
				this.ParentList.BucketList = this.ParentList.BucketList[1:]
			} else {
				bucketlist := make([]*RList, 0)
				bucketlist = append(bucketlist, this.ParentList.BucketList[:bucketIndex]...)
				bucketlist = append(bucketlist, this.ParentList.BucketList[bucketIndex+1:]...)
				this.ParentList.BucketList = bucketlist
			}

			for k, v := range this.IdList {
				//数据移动时高度不变化
				var vh *version.Height
				putValue := this.rootlist.putStub["__RLIST_"+this.RListName+"_DATAB_"+v]
				if putValue == nil {
					putValue, _ = this.rootlist.db.GetState(this.rootlist.ns, "__RLIST_"+this.RListName+"_DATAB_"+v)
				}

				if putValue != nil {
					vh = putValue.Version
				}

				//增加时不需要判断remove,高度不变
				this.rootlist.addIndexId(vh, (absouteIndex-index)+k, v, false)
			}
		}
	}
}

func (this *RList) getSaveData() *RListData {
	rdata := new(RListData)
	rdata.BucketId = this.BucketId
	rdata.BucketSeqNo = this.BucketSeqNo
	rdata.Size = int64(this.Size)
	rdata.IdList = this.IdList

	if this.ParentList != nil {
		rdata.ParentBucketId = this.ParentList.BucketId
	}

	var bucketlistData = make([]*BucketListData, 0)
	for _, v := range this.BucketList {
		bucket := new(BucketListData)
		bucket.BucketId = v.BucketId
		bucket.Size = int64(v.Size)
		bucketlistData = append(bucketlistData, bucket)
	}
	rdata.BucketListData = bucketlistData
	return rdata
}

func (this *RList) SaveState(vh *version.Height) {
	for _, v := range this.rootlist.mapRList {
		if v.needDelete {
			this.rootlist.putStub[v.BucketId] = nil
		} else if v.needSave {
			rdata := v.getSaveData()
			bData, _ := proto.Marshal(rdata)
			this.rootlist.putStub[v.BucketId] = &VersionedValue{bData, vh}
		}
	}
}

func (this *RList) GetPutStub() map[string]*VersionedValue {
	return this.putStub
}

func (this *RList) Reset() {
	this.rootlist.db = nil
	this.db = nil
	for k, v := range this.rootlist.mapRList {
		v.needSave = false

		if v.needDelete {
			delete(this.rootlist.mapRList, k)
		}
	}
	for k := range this.rootlist.putStub {
		delete(this.rootlist.putStub, k)
	}
}

func (this *RList) Clear() {
	this.rootlist.db = nil
	this.db = nil
	if this.mapRList == nil || len(this.mapRList) < 1 {
		return
	}

	for k, v := range this.mapRList {
		v.Clear()
		delete(this.mapRList, k)
	}

	this.mapRList = nil

	if this.rootlist.putStub != nil {
		for k := range this.rootlist.putStub {
			delete(this.rootlist.putStub, k)
		}

		this.rootlist.putStub = nil
	}

}

func indexOf(lisRList []*RList, theList *RList) int {
	nIndex := -1
	for k, v := range lisRList {
		if theList.BucketId == v.BucketId {
			nIndex = k
			break
		}
	}
	return nIndex
}
