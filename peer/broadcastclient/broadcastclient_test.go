package broadcastclient

import (
	"github.com/rongzer/blockchain/protos/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetCommunicateOrderer(t *testing.T) {
	bClient := GetCommunicateOrderer()
	assert.NotNil(t, bClient)
}

func TestBroadcastClient_SendToOrderer(t *testing.T) {
	bClient := GetCommunicateOrderer()
	rbcMessage := &common.RBCMessage{}
	resp, err := bClient.SendToOrderer(rbcMessage)
	assert.Error(t, err)
	assert.NotNil(t, resp)
}