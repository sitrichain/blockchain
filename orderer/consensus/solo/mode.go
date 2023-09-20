package solo

import (
	"github.com/rongzer/blockchain/orderer/consensus"
	cb "github.com/rongzer/blockchain/protos/common"
)

type mode struct{}

// Init 空初始化
func Init() consensus.Mode {
	return &mode{}
}

// NewConsenter 创建solo共识器
func (m *mode) NewConsenter(chain consensus.ChainResource, _ *cb.Metadata) (consensus.Consenter, error) {
	return newConsenter(chain), nil
}
