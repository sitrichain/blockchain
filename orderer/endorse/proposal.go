package endorse

import (
	"fmt"
	"time"

	"github.com/rongzer/blockchain/common/log"
	pmsp "github.com/rongzer/blockchain/protos/msp"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

// proposal 提案数据
type proposal struct {
	time        time.Time         // 接收时间
	proposal    *pb.Proposal      // 提案
	attachs     map[string]string // 附件
	sourcePeer  string            // 来源地址
	endorserNum int               // 背书发送数
	resPayload  []byte            // 背书回复消息payload
	endorsers   []*pb.Endorsement // 背书者信息
}

// 获取提案信息
func (p *proposal) getInfo() string {
	hdr, err := utils.GetHeader(p.proposal.Header)
	if err != nil {
		log.Logger.Errorf("EndorserProposal message err:%s", err)
		return ""
	}

	hdrExt, err := utils.GetChaincodeHeaderExtension(hdr)
	if err != nil {
		log.Logger.Errorf("EndorserProposal message err:%s", err)
		return ""
	}

	cis, err := utils.GetChaincodeInvocationSpec(p.proposal)
	if err != nil {
		log.Logger.Errorf("get ChaincodeInvocationSpec from proposal err:%s", err)
		return ""
	}

	args := cis.ChaincodeSpec.Input.Args
	txInfo := ""
	for _, arg := range args {
		if len(txInfo) < 1 {
			txInfo += string(arg)
		} else {
			txInfo += "," + string(arg)
		}
	}

	shdr, err := utils.GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		log.Logger.Errorf("get shdr from SignatureHeader err:%s", err)
		return ""
	}
	serializedIdentity := &pmsp.SerializedIdentity{}
	if err = serializedIdentity.Unmarshal(shdr.Creator); err != nil {
		log.Logger.Errorf("get Creator from Header err:%s", err)
		return ""
	}
	return fmt.Sprintf("chaincodeName:%s args:%s \n cert:\n%s", hdrExt.ChaincodeId.Name, txInfo, string(serializedIdentity.IdBytes))
}
