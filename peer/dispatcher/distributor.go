package dispatcher

import (
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/endorserclient"
	"github.com/rongzer/blockchain/peer/statistics"
	"github.com/rongzer/blockchain/protos/common"
)

type Distributor interface {
	SendToEndorse(message *common.RBCMessage) (*common.RBCMessage, []string, error)
	MarkProposalFail(message *common.RBCMessage) (string, bool)
	AddUpEndorseSuccess(message *common.RBCMessage)
	GetUnendorseCount() int
	MarshalPeerList(chain string) ([]byte, error)
	GetClientManager() *endorserclient.Manager
}

// NewDistributor 创建背书分配器
func NewDistributor(clientManager *endorserclient.Manager, chainManager *chain.Manager, signer crypto.LocalSigner) Distributor {
	go statistics.LoopPrint()
	return NewRaftDistributor(clientManager, chainManager, signer)
}
