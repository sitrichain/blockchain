package raft

import (
	"github.com/golang/protobuf/proto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

// blockCreator holds number and hash of latest block
// so that next block will be created based on it.
type blockCreator struct {
	hash   []byte
	number uint64

	logger *log.RaftLogger
}

func (bc *blockCreator) createNextBlock(envs []*common.Envelope) *common.Block {
	data := &common.BlockData{
		Data: make([][]byte, len(envs)),
	}

	var err error
	for i, env := range envs {
		data.Data[i], err = proto.Marshal(env)
		if err != nil {
			bc.logger.Panicf("Could not marshal envelope: %s", err)
		}
	}

	bc.number++

	block := common.NewBlock(bc.number, bc.hash)
	block.Header.DataHash = utils.BlockDataHash(data)
	block.Data = data

	bc.hash = utils.BlockHeaderHash(block.Header)
	return block
}
