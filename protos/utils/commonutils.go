/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

                 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rongzer/blockchain/common/conf"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/rongzer/blockchain/common/crypto"
	cb "github.com/rongzer/blockchain/protos/common"
	pb "github.com/rongzer/blockchain/protos/peer"
)

// MarshalOrPanic serializes a protobuf message and panics if this operation fails.
func MarshalOrPanic(pb proto.Message) []byte {
	data, err := proto.Marshal(pb)
	if err != nil {
		panic(err)
	}
	return data
}

// Marshal serializes a protobuf message.
func Marshal(pb proto.Message) ([]byte, error) {
	return proto.Marshal(pb)
}

// CreateNonceOrPanic generates a nonce using the common/crypto package
// and panics if this operation fails.
func CreateNonceOrPanic() []byte {
	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		panic(fmt.Errorf("Cannot generate random nonce: %s", err))
	}
	return nonce
}

// UnmarshalPayload unmarshals bytes to a Payload structure
func UnmarshalPayload(encoded []byte) (*cb.Payload, error) {
	payload := &cb.Payload{}
	err := payload.Unmarshal(encoded)
	if err != nil {
		return nil, err
	}
	return payload, err
}

// UnmarshalEnvelope unmarshals bytes to an Envelope structure
func UnmarshalEnvelope(encoded []byte) (*cb.Envelope, error) {
	envelope := &cb.Envelope{}
	err := envelope.Unmarshal(encoded)
	if err != nil {
		return nil, err
	}
	return envelope, err
}

// UnmarshalEnvelopeOfType unmarshals an envelope of the specified type, including
// the unmarshaling the payload data
func UnmarshalEnvelopeOfType(envelope *cb.Envelope, headerType cb.HeaderType, message proto.Message) (*cb.ChannelHeader, error) {
	payload, err := UnmarshalPayload(envelope.Payload)
	if err != nil {
		return nil, err
	}

	if payload.Header == nil {
		return nil, fmt.Errorf("Envelope must have a Header")
	}

	chdr, err := UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, fmt.Errorf("Invalid ChannelHeader")
	}

	if chdr.Type != int32(headerType) {
		return nil, fmt.Errorf("Not a tx of type %v", headerType)
	}
	if err = proto.Unmarshal(payload.Data, message); err != nil {
		return nil, fmt.Errorf("Error unmarshaling message for type %v: %s", headerType, err)
	}

	return chdr, nil
}

// ChannelHeader returns the *cb.ChannelHeader for a given *cb.Envelope.
func ChannelHeader(env *cb.Envelope) (*cb.ChannelHeader, error) {
	envPayload, err := UnmarshalPayload(env.Payload)
	if err != nil {
		return nil, err
	}

	if envPayload.Header == nil {
		return nil, errors.New("header not set")
	}

	if envPayload.Header.ChannelHeader == nil {
		return nil, errors.New("channel header not set")
	}

	chdr, err := UnmarshalChannelHeader(envPayload.Header.ChannelHeader)
	if err != nil {
		return nil, errors.WithMessage(err, "error unmarshaling channel header")
	}

	return chdr, nil
}

// ExtractEnvelope retrieves the requested envelope from a given block and unmarshals it.
func ExtractEnvelope(block *cb.Block, index int) (*cb.Envelope, error) {
	if block.Data == nil {
		return nil, fmt.Errorf("No data in block")
	}

	envelopeCount := len(block.Data.Data)
	if index < 0 || index >= envelopeCount {
		return nil, fmt.Errorf("Envelope index out of bounds")
	}
	marshaledEnvelope := block.Data.Data[index]
	envelope, err := GetEnvelopeFromBlock(marshaledEnvelope)
	if err != nil {
		return nil, fmt.Errorf("Block data does not carry an envelope at index %d: %s", index, err)
	}
	return envelope, nil
}

// MakeChannelHeader creates a ChannelHeader.
func MakeChannelHeader(headerType cb.HeaderType, chainID string) *cb.ChannelHeader {
	return &cb.ChannelHeader{
		Type:    int32(headerType),
		Version: 0,
		Timestamp: &types.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   0,
		},
		ChannelId: chainID,
		Epoch:     0,
	}
}

// MakeSignatureHeader creates a SignatureHeader.
func MakeSignatureHeader(serializedCreatorCertChain []byte, nonce []byte) *cb.SignatureHeader {
	return &cb.SignatureHeader{
		Creator: serializedCreatorCertChain,
		Nonce:   nonce,
	}
}

func SetTxID(channelHeader *cb.ChannelHeader, signatureHeader *cb.SignatureHeader) error {
	txid, err := ComputeProposalTxID(
		signatureHeader.Nonce,
		signatureHeader.Creator,
	)
	if err != nil {
		return err
	}
	channelHeader.TxId = txid
	return nil
}

// MakePayloadHeader creates a Payload Header.
func MakePayloadHeader(ch *cb.ChannelHeader, sh *cb.SignatureHeader) *cb.Header {
	return &cb.Header{
		ChannelHeader:   MarshalOrPanic(ch),
		SignatureHeader: MarshalOrPanic(sh),
	}
}

// UnmarshalChannelHeader returns a ChannelHeader from bytes
func UnmarshalChannelHeader(bytes []byte) (*cb.ChannelHeader, error) {
	chdr := &cb.ChannelHeader{}
	err := chdr.Unmarshal(bytes)
	if err != nil {
		return nil, fmt.Errorf("UnmarshalChannelHeader failed, err %s", err)
	}

	return chdr, nil
}

// UnmarshalChaincodeID returns a ChaincodeID from bytes
func UnmarshalChaincodeID(bytes []byte) (*pb.ChaincodeID, error) {
	ccid := &pb.ChaincodeID{}
	if err := ccid.Unmarshal(bytes); err != nil {
		return nil, fmt.Errorf("UnmarshalChaincodeID failed, err %s", err)
	}

	return ccid, nil
}

// IsConfigBlock validates whenever given block contains configuration
// update transaction
func IsConfigBlock(block *cb.Block) bool {
	envelope, err := ExtractEnvelope(block, 0)
	if err != nil {
		return false
	}

	payload, err := GetPayload(envelope)
	if err != nil {
		return false
	}

	if payload.Header == nil {
		return false
	}

	hdr, err := UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return false
	}

	return cb.HeaderType(hdr.Type) == cb.HeaderType_CONFIG
}

// ConfigEnvelopeFromBlock extracts configuration envelope from the block based on the
// config type, i.e. HeaderType_ORDERER_TRANSACTION or HeaderType_CONFIG
func GetConfigEnvelopeFromBlock(block *cb.Block) (*cb.Envelope, int32, bool, error) {
	if block == nil {
		return nil, 0, false, errors.New("nil block")
	}

	envelope, err := ExtractEnvelope(block, 0)
	if err != nil {
		return nil, 0, false, errors.Wrapf(err, "failed to extract envelope from the block")
	}

	channelHeader, err := ChannelHeader(envelope)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "cannot extract channel header")
	}

	switch channelHeader.Type {
	case int32(cb.HeaderType_ORDERER_TRANSACTION):
		var ableToCreateChain bool
		// 从attach中抽取firstNode，若等于自身节点地址，说明自身节点为该链的创建者，也是该链的raft集群的第一个节点，返回true
		if envelope.Attachs != nil && len(envelope.Attachs) > 0 {
			firstNode := envelope.Attachs["firstNode"]
			if firstNode == conf.V.Sealer.Raft.EndPoint {
				ableToCreateChain = true
			}
		}
		payload, err := UnmarshalPayload(envelope.Payload)
		if err != nil {
			return nil, 0, false, errors.Wrap(err, "failed to unmarshal envelope to extract config payload for orderer transaction")
		}
		configEnvelop, err := UnmarshalEnvelope(payload.Data)
		if err != nil {
			return nil, 0, false, errors.Wrap(err, "failed to unmarshal config envelope for orderer type transaction")
		}

		return configEnvelop, 4, ableToCreateChain, nil
	case int32(cb.HeaderType_CONFIG):
		return envelope, 1, false, nil
	default:
		return nil, 0, false, errors.Errorf("unexpected header type: %v", channelHeader.Type)
	}
}
