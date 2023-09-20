package endorser

import (
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp"
	"github.com/rongzer/blockchain/peer/broadcastclient"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/chaincode"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/endorser/executor"
	"github.com/rongzer/blockchain/peer/endorser/validator"
	"github.com/rongzer/blockchain/peer/events/producer"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/protos/common"
	pmsp "github.com/rongzer/blockchain/protos/msp"
	ab "github.com/rongzer/blockchain/protos/orderer"
	pb "github.com/rongzer/blockchain/protos/peer"
	putils "github.com/rongzer/blockchain/protos/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

const (
	EndorseSuccess = 5
	EndorseFailed  = 6
)

// >>>>> begin errors section >>>>>
//chaincodeError is a blockchain error signifying error from chaincode
type chaincodeError struct {
	status int32
	msg    string
}

func (ce chaincodeError) Error() string {
	return fmt.Sprintf("chaincode error (status: %d, message: %s)", ce.status, ce.msg)
}

// Endorser provides the Endorser service ProcessProposal
type Endorser struct {
	validator *validator.Validator
	executor  *executor.Executor
	// 从orderer过来的背书通道
	endorserChan chan *common.RBCMessage
	peerAddress  string
}

// NewEndorserServer creates and returns a new Endorser server instance.
func NewEndorserServer() pb.EndorserServer {
	e := new(Endorser)

	e.validator = validator.NewValidator()
	e.executor = executor.NewExecutor()
	e.endorserChan = make(chan *common.RBCMessage, 5000)
	e.peerAddress = viper.GetString("peer.address")

	go e.execEndorser()
	return e
}

func (*Endorser) getTxSimulator(ledgername string) (ledger.TxSimulator, error) {
	lgr := chain.GetLedger(ledgername)
	if lgr == nil {
		return nil, fmt.Errorf("channel does not exist: %s", ledgername)
	}
	return lgr.NewTxSimulator()
}

func (*Endorser) getHistoryQueryExecutor(ledgername string) (ledger.HistoryQueryExecutor, error) {
	lgr := chain.GetLedger(ledgername)
	if lgr == nil {
		return nil, fmt.Errorf("channel does not exist: %s", ledgername)
	}
	return lgr.NewHistoryQueryExecutor()
}

// Only exposed for testing purposes - commit the tx simulation so that
// a deploy transaction is persisted and that chaincode can be invoked.
// This makes the endorser test self-sufficient
func (e *Endorser) commitTxSimulation(proposal *pb.Proposal, chainID string, signer msp.SigningIdentity, pResp *pb.ProposalResponse, blockNumber uint64) error {
	tx, err := putils.CreateSignedTx(proposal, signer, pResp)
	if err != nil {
		return err
	}

	lgr := chain.GetLedger(chainID)
	if lgr == nil {
		return fmt.Errorf("failure while looking up the ledger")
	}

	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return err
	}
	block := common.NewBlock(blockNumber, []byte{})
	block.Data.Data = [][]byte{txBytes}
	block.Header.DataHash = block.Data.Hash()
	if err = lgr.Commit(block); err != nil {
		return err
	}

	return nil
}

// ProcessProposal process the Proposal
func (e *Endorser) ProcessProposal(ctx context.Context, signedProp *pb.SignedProposal) (*pb.ProposalResponse, error) {

	// at first, we check whether the message is valid
	prop, chdr, _, hdrExt, err := e.validator.ValidateEndorserProposal(signedProp)
	if err != nil {
		return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
	}

	chainID := chdr.ChannelId
	txid := chdr.TxId

	// obtaining once the tx simulator for this proposal. This will be nil
	// for chainless proposals
	// Also obtain a history query executor for history queries, since tx simulator does not cover history
	var txsim ledger.TxSimulator
	var historyQueryExecutor ledger.HistoryQueryExecutor
	if chainID != "" {
		if txsim, err = e.getTxSimulator(chainID); err != nil {
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
		}
		if historyQueryExecutor, err = e.getHistoryQueryExecutor(chainID); err != nil {
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
		}
		// Add the historyQueryExecutor to context
		// TODO shouldn't we also add txsim to context here as well? Rather than passing txsim parameter
		// around separately, since eventually it gets added to context anyways
		ctx = context.WithValue(ctx, chaincode.HistoryQueryExecutorKey, historyQueryExecutor)

		defer txsim.Done()
	}
	//this could be a request to a chainless SysCC

	//1 -- simulate
	cd, res, simulationResult, ccevent, err := e.executor.SimulateProposal(ctx, chainID, txid, signedProp, prop, hdrExt.ChaincodeId, txsim)
	if err != nil {
		return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
	}
	if res != nil {
		if res.Status >= shim.ERROR {
			var cceventBytes []byte
			if ccevent != nil {
				cceventBytes, err = putils.GetBytesChaincodeEvent(ccevent)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal event bytes - %s", err)
				}
			}
			pResp, err := putils.CreateProposalResponseFailure(prop.Header, prop.Payload, res, simulationResult,
				cceventBytes, hdrExt.ChaincodeId, hdrExt.PayloadVisibility)
			if err != nil {
				return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
			}

			return pResp, &chaincodeError{res.Status, res.Message}
		}
	}

	//2 -- endorse and get a marshalled ProposalResponse message
	var pResp *pb.ProposalResponse

	//TODO till we implement global ESCC, CSCC for system chaincodes
	//chainless proposals (such as CSCC) don't have to be endorsed
	if chainID == "" {
		pResp = &pb.ProposalResponse{Response: res}
	} else {
		pResp, err = e.executor.EndorseProposal(ctx, chainID, txid, signedProp, prop, res, simulationResult,
			ccevent, hdrExt.PayloadVisibility, hdrExt.ChaincodeId, txsim, cd)
		if err != nil {
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
		}
		if pResp != nil {
			if res.Status >= shim.ERRORTHRESHOLD {
				return pResp, &chaincodeError{res.Status, res.Message}
			}
		}
	}

	// Set the proposal response payload - it
	// contains the "return value" from the
	// chaincode invocation
	pResp.Response.Payload = res.Payload

	return pResp, nil
}

// 接收交易，交通过orderer分发
func (e *Endorser) SendTransaction(_ context.Context, rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {
	logger := log.Logger.With(zap.String("TxID", rbcMessage.TxID), zap.Int32("type", rbcMessage.Type))
	defer func() {
		if err := recover(); err != nil {
			logger.Error(err)
		}
	}()

	switch rbcMessage.Type {
	case 0:
		return e.createChain(logger, rbcMessage)
	case 1:
		return e.getBlock(logger, rbcMessage)
	case 3:
		return e.sendEndorsement(logger, rbcMessage)
	case 4:
		return e.receiveEndorsement(rbcMessage)
	case 7:
		return e.endorseResponse(logger, rbcMessage)
	case 21:
		return e.sendHttpRequest(logger, rbcMessage)
	case 23:
		return e.getUnendorsedCount(logger, rbcMessage)
	case 25:
		return e.handleDeliverMessage(logger, rbcMessage)
	default:
		return &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}}, nil
	}
}

func (e *Endorser) createChain(logger *zap.SugaredLogger, rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {
	logger.Debug("createChain with rbcmessage :  ", rbcMessage)
	// 发送给Orderer
	res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
	if err != nil {
		log.Logger.Errorf("Rejecting createChain message because of send to orderer error. %s", err)
		return nil, err
	}

	buf, err := res.Marshal()
	if err != nil {
		log.Logger.Errorf("Rejecting createChain message because of marshal result error. %s", err)
		return nil, err
	}

	return &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}, Payload: buf}, nil
}

func (e *Endorser) getBlock(logger *zap.SugaredLogger, rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {
	logger.Debug("getBlock with rbcmessage :  ", rbcMessage)

	lgr := chain.GetLedger(rbcMessage.ChainID)
	if lgr != nil {
		bnum := uint64(0)
		if rbcMessage.Extend == "LAST" {
			blockchainInfo, _ := lgr.GetBlockchainInfo()
			bnum = blockchainInfo.Height
		} else {
			bnum, _ = strconv.ParseUint(rbcMessage.Extend, 10, 64)
		}

		if bnum >= 0 {
			block, err := lgr.GetBlockByNumber(bnum)
			if block != nil && err == nil {
				blockBuf, err := block.Marshal()
				if err == nil && blockBuf != nil { //block异常
					bcRes := &ab.BroadcastResponse{Status: common.Status_SUCCESS, Data: blockBuf}
					buf, err := bcRes.Marshal()
					if err == nil && buf != nil {
						return &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}, Payload: buf}, nil
					}
				}
			}
		}
	}

	// 发送给Orderer
	res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
	if err != nil {
		log.Logger.Errorf("Rejecting getBlock message because send to orderer error %s", err)
		return nil, err
	}

	buf, err := res.Marshal()
	if err != nil {
		log.Logger.Errorf("Rejecting getBlock message because marshal result error %s", err)
		return nil, err
	}

	return &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}, Payload: buf}, nil
}

func (e *Endorser) sendEndorsement(logger *zap.SugaredLogger, rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {
	logger.Debug("sendEndorsement with rbcmessage :  ", rbcMessage)

	response := &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}}
	//原始消息，在extend上带上发送者地址
	rbcMessage.Extend = e.peerAddress

	var buf []byte
	var err error
	res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	buf, err = res.Marshal()
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	response.Payload = buf
	return response, nil
}

func (e *Endorser) receiveEndorsement(rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {

	response := &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}}
	e.endorserChan <- rbcMessage

	return response, nil
}

func (e *Endorser) endorseResponse(logger *zap.SugaredLogger, rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {

	response := &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}}
	var err error
	if string(rbcMessage.Data) == "success" {
		err = producer.Send(producer.CreateSuccessEvent(rbcMessage.ChainID, rbcMessage.TxID))
	} else {
		err = producer.Send(producer.CreateRejectionEvent(rbcMessage.ChainID, rbcMessage.TxID, string(rbcMessage.Data)))
	}

	if err != nil {
		logger.Errorf("producer send err : %s", err)
		return nil, err
	}

	return response, nil
}

func (e *Endorser) sendHttpRequest(logger *zap.SugaredLogger, rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {

	response := &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}}

	res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
	if err != nil {
		logger.Errorf("send exec transaction to orderer %v,err:%s", res, err)
		return nil, err
	}
	buf, err := res.Marshal()
	if err != nil {
		logger.Errorf("send exec transaction to orderer %v,err:%s", res, err)
		return nil, err
	}
	response.Payload = buf

	return response, nil
}

func (e *Endorser) getUnendorsedCount(logger *zap.SugaredLogger, rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {

	response := &pb.ProposalResponse{Response: &pb.Response{Status: 200, Message: "send transaction success"}}

	res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
	if err != nil {
		logger.Errorf("send exec transaction to orderer %v,err:%s", res, err)
		return nil, err
	}
	response.Payload = res.Data

	return response, nil
}

func (e *Endorser) handleDeliverMessage(logger *zap.SugaredLogger, rbcMessage *common.RBCMessage) (*pb.ProposalResponse, error) {
	signProp := &pb.SignedProposal{}
	if err := signProp.Unmarshal(rbcMessage.Data); err != nil {
		logger.Error(err)
		return nil, err
	}

	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()

	return e.ProcessProposal(ctx, signProp)
}

func (e *Endorser) execEndorser() {
	eNum := viper.GetInt("peer.endorser.num")
	if eNum < 1 || eNum > 100 {
		eNum = 10
	}

	ch := make(chan int, eNum)
	for {
		select {
		case rbcMessage := <-e.endorserChan:
			ch <- 1
			e.EndorserProposal(rbcMessage, ch)
		}
	}
}

// ProcessProposal process the Proposal
func (e *Endorser) EndorserProposal(rbcMessage *common.RBCMessage, ch chan int) {
	defer func() {
		if err := recover(); err != nil {
			log.Logger.Error(err)
		}
	}()

	signedProp := &pb.SignedProposal{}
	if err := signedProp.Unmarshal(rbcMessage.Data); err != nil {
		<-ch
		log.Logger.Error(err)
		return
	}

	prop, chdr, shdr, hdrExt, err := e.validator.ValidateEndorserProposal(signedProp)
	if err != nil {
		<-ch
		log.Logger.Error(err)
		return
	}

	runStateKey := rbcMessage.ChainID + ":" + hdrExt.ChaincodeId.Name
	runStatus, _ := chaincode.EndorserChan.Load(runStateKey)
	//正常运行中
	if runStatus == 1 {
		go e.processEndorser(rbcMessage, signedProp, prop, chdr, shdr, hdrExt, ch)
	} else {
		e.processEndorser(rbcMessage, signedProp, prop, chdr, shdr, hdrExt, ch)
	}

}

func (e *Endorser) processEndorser(rbcMessage *common.RBCMessage, signedProp *pb.SignedProposal, prop *pb.Proposal,
	chdr *common.ChannelHeader, shdr *common.SignatureHeader, hdrExt *pb.ChaincodeHeaderExtension, ch chan int) {

	res, err := e.processEndorserExec(signedProp, prop, chdr, hdrExt)

	<-ch

	var retMessage *common.RBCMessage

	if err != nil {
		errInfo := err.Error()
		if errInfo != "TransactionExist" {
			//发送错误信息至endorser
			cis, err1 := putils.GetChaincodeInvocationSpec(prop)
			if err1 != nil {
				log.Logger.Error(err1)
				return
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

			serializedIdentity := &pmsp.SerializedIdentity{}
			err1 = serializedIdentity.Unmarshal(shdr.Creator)
			if err1 != nil {
				log.Logger.Error(err1)
				return
			}
			txInfo = fmt.Sprintf("TxID:%s,chaincodeName:%s args:%s \n cert:\n%s", rbcMessage.TxID, hdrExt.ChaincodeId.Name,
				txInfo, string(serializedIdentity.IdBytes))
		}
		//rbc执行错误反馈
		retMessage = &common.RBCMessage{Type: EndorseFailed, Data: []byte(errInfo), ChainID: rbcMessage.ChainID, TxID: rbcMessage.TxID}
	} else {
		key := rbcMessage.ChainID + ":" + hdrExt.ChaincodeId.Name
		chaincode.EndorserChan.Store(key, 1)
	}

	if res != nil && retMessage == nil {
		buf, err := res.Marshal()
		if err != nil {
			log.Logger.Error(err)
			return
		}
		retMessage = &common.RBCMessage{Type: EndorseSuccess, Data: buf, ChainID: rbcMessage.ChainID, TxID: rbcMessage.TxID}
	}

	_, err = broadcastclient.GetCommunicateOrderer().SendToOrderer(retMessage)
	if err != nil {
		log.Logger.Error(err)
		return
	}

}

// ProcessProposal process the Proposal
func (e *Endorser) processEndorserExec(signedProp *pb.SignedProposal, prop *pb.Proposal, chdr *common.ChannelHeader,
	hdrExt *pb.ChaincodeHeaderExtension) (*pb.ProposalResponse, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error

	chainID := chdr.ChannelId
	txid := chdr.TxId

	// obtaining once the tx simulator for this proposal. This will be nil
	// for chainless proposals
	// Also obtain a history query executor for history queries, since tx simulator does not cover history
	var txsim ledger.TxSimulator
	var historyQueryExecutor ledger.HistoryQueryExecutor
	if chainID != "" {
		if txsim, err = e.getTxSimulator(chainID); err != nil {
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
		}
		if historyQueryExecutor, err = e.getHistoryQueryExecutor(chainID); err != nil {
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
		}
		// Add the historyQueryExecutor to context
		// TODO shouldn't we also add txsim to context here as well? Rather than passing txsim parameter
		// around separately, since eventually it gets added to context anyways
		ctx = context.WithValue(ctx, chaincode.HistoryQueryExecutorKey, historyQueryExecutor)

		defer txsim.Done()
	}
	//this could be a request to a chainless SysCC

	//1 -- simulate
	cd, res, simulationResult, ccevent, err := e.executor.SimulateProposal(ctx, chainID, txid, signedProp, prop, hdrExt.ChaincodeId, txsim)
	if err != nil {
		return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
	}

	if res != nil {
		if res.Status >= shim.ERROR {
			var cceventBytes []byte
			if ccevent != nil {
				cceventBytes, err = putils.GetBytesChaincodeEvent(ccevent)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal event bytes - %s", err)
				}
			}
			pResp, err := putils.CreateProposalResponseFailure(prop.Header, prop.Payload, res, simulationResult,
				cceventBytes, hdrExt.ChaincodeId, hdrExt.PayloadVisibility)
			if err != nil {
				return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
			}

			return pResp, &chaincodeError{res.Status, res.Message}
		}
	}

	//2  -- endorse and get a marshalled ProposalResponse message
	var pResp *pb.ProposalResponse

	//TODO till we implement global ESCC, CSCC for system chaincodes
	//chainless proposals (such as CSCC) don't have to be endorsed
	if chainID == "" {
		pResp = &pb.ProposalResponse{Response: res}
	} else {
		pResp, err = e.executor.EndorseProposal(ctx, chainID, txid, signedProp, prop, res, simulationResult,
			ccevent, hdrExt.PayloadVisibility, hdrExt.ChaincodeId, txsim, cd)
		if err != nil {
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, err
		}
		if pResp != nil {
			if res.Status >= shim.ERRORTHRESHOLD {
				return pResp, &chaincodeError{res.Status, res.Message}
			}
		}
	}

	// Set the proposal response payload - it
	// contains the "return value" from the
	// chaincode invocation
	pResp.Response.Payload = res.Payload

	return pResp, nil
}
