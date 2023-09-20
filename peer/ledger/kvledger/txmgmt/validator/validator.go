package validator

import (
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/statedb"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/peer"
)

// Validator 验证器接口, 用于验证读写集
type Validator interface {
	ValidateAndPrepareBatch(block *common.Block, doMVCCValidation bool) (*statedb.UpdateBatch, error)
	ValidateEndorserTX(envBytes []byte, doMVCCValidation bool, updates *statedb.UpdateBatch) (*rwsetutil.TxRwSet, peer.TxValidationCode, error)
}
