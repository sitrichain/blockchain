package dispatcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/peer/consensus/raft"
	"github.com/rongzer/blockchain/peer/endorserclient"
	"sync"
	"time"

	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/statistics"
	cb "github.com/rongzer/blockchain/protos/common"
	wrset "github.com/rongzer/blockchain/protos/ledger/rwset"
	"github.com/rongzer/blockchain/protos/ledger/rwset/kvrwset"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

var raftDistributor *RaftDistributor

func GetRaftDistributor() *RaftDistributor {
	return raftDistributor
}

func NewRaftDistributor(clientManager *endorserclient.Manager, chainManager *chain.Manager, signer crypto.LocalSigner) *RaftDistributor {
	peers := make(map[string]*RaftPeerList)
	chainManager.Chains.Range(func(k, v interface{}) bool {
		chainId := k.(string)
		chain, _ := chainManager.GetChain(chainId)
		peers[chainId] = newRaftPeerList(chain.Consenter.(*raft.Consenter), chainId)
		return true
	})
	raftDistributor = &RaftDistributor{
		clientManager: clientManager,
		chainManager:  chainManager,
		signer:        signer,
		proposals:     make(map[string]*proposal),
		peers:         peers,
		attachClient:  comm.NewHttpClient(),
	}
	// 定时更新raftDistributor.peers
	go func() {
		t := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-t.C:
				updatePeers(chainManager)
			}
		}
	}()
	return raftDistributor
}

func updatePeers(chainManager *chain.Manager) {
	chainManager.Chains.Range(func(k, v interface{}) bool {
		chainId := k.(string)
		if _, ok := raftDistributor.peers[chainId]; ok {
			return false
		}
		chain, _ := chainManager.GetChain(chainId)
		raftDistributor.peers[chainId] = newRaftPeerList(chain.Consenter.(*raft.Consenter), chainId)
		return true
	})

}

// RaftDistributor 背书分配器
type RaftDistributor struct {
	clientManager *endorserclient.Manager
	chainManager  *chain.Manager
	signer        crypto.LocalSigner
	proposalsLock sync.Mutex
	proposals     map[string]*proposal
	peersLock     sync.Mutex
	peers         map[string]*RaftPeerList
	// 添加httpclient
	attachClient *comm.HttpClient
}

func (d *RaftDistributor) GetClientManager() *endorserclient.Manager {
	return d.clientManager
}

func (d *RaftDistributor) MarshalPeerList(chain string) ([]byte, error) {
	d.peersLock.Lock()
	l, ok := d.peers[chain]
	d.peersLock.Unlock()
	if !ok {
		return nil, fmt.Errorf("no such chain %s in distributor", chain)
	}
	if len(l.peers.List) == 0 {
		return nil, fmt.Errorf("no peer participated in endorsement under this chain: %v", chain)
	}
	return l.marshal()
}

// SendToEndorse 分发背书
func (d *RaftDistributor) SendToEndorse(message *cb.RBCMessage) (*cb.RBCMessage, []string, error) {
	// 创建背书请求消息
	message.Type = 4
	signedProp := &pb.SignedProposal{}
	if err := signedProp.Unmarshal(message.Data); err != nil {
		log.Logger.Errorf("distribute tx %s fail because unmarshal signed proposal err: %s", message.TxID, err)
		return nil, nil, err
	}
	prop, err := utils.GetProposal(signedProp.ProposalBytes)
	if err != nil {
		log.Logger.Errorf("distribute tx %s fail because get Proposal from signed proposal err: %s", message.TxID, err)
		return nil, nil, err
	}
	sourcePeer := message.Extend
	// Extend字段记录是哪个节点分发的这条消息去背书
	message.Extend = conf.V.Sealer.Raft.EndPoint
	// 记录提案
	proposal := &proposal{
		time:        time.Now(),
		proposal:    prop,
		attachs:     signedProp.Attachs,
		sourcePeer:  sourcePeer,
		endorserNum: 0,
		endorsers:   make([]*pb.Endorsement, 0)}

	// 不存在该提案记录时, 记录该提案
	d.proposalsLock.Lock()
	if _, ok := d.proposals[message.TxID]; ok {
		d.proposalsLock.Unlock()
		return nil, nil, err
	}
	d.proposals[message.TxID] = proposal
	d.proposalsLock.Unlock()

	// 选取节点分发
	var target []string
	// 判断链码信息, rbcapproval时只用找来源节点背书
	cis, err := utils.GetChaincodeInvocationSpec(prop)
	if err != nil {
		log.Logger.Errorf("distribute tx %s fail because get ChaincodeInvocationSpec from proposal err:%s", message.TxID, err)
		return nil, nil, err
	}
	num := 1
	args := cis.ChaincodeSpec.Input.Args
	if args != nil && len(args) >= 2 && cis.ChaincodeSpec.ChaincodeId.Name == "rbcapproval" && string(args[1]) == "newApproval" {
		// 只发送给来源节点背书
		target = []string{sourcePeer}
	} else {
		// 从负载均衡选出一些节点
		var p *RaftPeerList
		for i := 0; i < 5; i++ {
			d.peersLock.Lock()
			v, ok := d.peers[message.ChainID]
			d.peersLock.Unlock()
			if !ok {
				time.Sleep(time.Second * 2)
			} else {
				p = v
				break
			}
		}
		if p == nil {
			err = fmt.Errorf("distribute tx %s fail because not found chain %s", message.TxID, message.ChainID)
			log.Logger.Error(err)
			return nil, nil, err
		}
		// 多次重试
		for i := 0; i < 5; i++ {
			target, num = p.balancer.pick()
			if num == 0 {
				time.Sleep(time.Second * 2)
			} else {
				break
			}
		}
		if num == 0 {
			// 若共识器尚未准备好，即无法分发背书，则自己背书。这种情况只会出现在——节点以learner身份同步区块(catchup)时
			proposal.endorserNum = 1
			statistics.SentEndorseCount.Inc()
			return message, []string{conf.V.Peer.Endpoint}, nil
			//err = fmt.Errorf("distribute tx %s fail because not endorser peers pick from balancer", message.TxID)
			//log.Logger.Error(err)
			//return nil, nil, err
		}
		log.Logger.Debugf("TX %s pick endorser %v from %v", message.TxID, target, p.balancer.all())
	}
	//// 设置背书采集数量是分发数量-1
	//if num > 1 {
	//	num--
	//}
	proposal.endorserNum = num
	// 发送背书请求
	statistics.SentEndorseCount.Inc()
	return message, target, nil
}

// MarkEndorseFail 标记背书失败,返回提案信息
func (d *RaftDistributor) MarkProposalFail(message *cb.RBCMessage) (string, bool) {
	d.proposalsLock.Lock()
	defer d.proposalsLock.Unlock()
	p, ok := d.proposals[message.TxID]
	if !ok {
		return "", false
	}
	delete(d.proposals, message.TxID)

	// 发送背书错误通知至来源Peer
	message.Type = 7
	go d.clientManager.SendToPeers([]string{p.sourcePeer}, message)

	return p.getInfo(), true
}

// AddUpEndorseSuccess 累计背书成功消息
func (d *RaftDistributor) AddUpEndorseSuccess(message *cb.RBCMessage) {
	d.proposalsLock.Lock()
	defer d.proposalsLock.Unlock()
	proposalRequest, ok := d.proposals[message.TxID]
	if !ok {
		return
	}

	proposalResponse := &pb.ProposalResponse{}
	if err := proposalResponse.Unmarshal(message.Data); err != nil {
		log.Logger.Errorf("Add up endorse success fail because unmarshal ProposalResponse err : %s", err)
		return
	}

	for _, endorsement := range proposalRequest.endorsers {
		// 已有该背书者的背书存在
		if bytes.Equal(endorsement.Endorser, proposalResponse.Endorsement.Endorser) {
			return
		}
	}
	// 增加背书者记录
	proposalRequest.endorsers = append(proposalRequest.endorsers, proposalResponse.Endorsement)

	// 处理有足够累计背书的提案
	if proposalRequest.endorserNum <= len(proposalRequest.endorsers) {
		delete(d.proposals, message.TxID)
		proposalRequest.resPayload = proposalResponse.Payload

		// 提案转交易, 然后排序落块
		go d.makeEnvelopeToOrder(proposalRequest, message.ChainID, message.TxID)

		// 通知提案来源节点背书成功
		sucMessage := &cb.RBCMessage{Type: 7, Data: []byte("success"), ChainID: message.ChainID, TxID: message.TxID}
		go d.clientManager.SendToPeers([]string{proposalRequest.sourcePeer}, sucMessage)

	}
}

// GetUnendorseCount 获取未背书完成提案的数量
func (d *RaftDistributor) GetUnendorseCount() int {
	d.proposalsLock.Lock()
	defer d.proposalsLock.Unlock()
	return len(d.proposals)
}

// 获取提案背书结果中是否有写集
func (d *RaftDistributor) writeSetIsEmpty(request *proposal) (bool, error) {
	// 从提案中获取读写集
	proposalResponsePayload, err := utils.GetProposalResponsePayload(request.resPayload)
	if err != nil {
		log.Logger.Errorf("Make proposal to transaction fail because parse ProposalResponsePayload err when check write set: %s", err)
		return true, err
	}
	chaincodeAction := &pb.ChaincodeAction{}
	if err = chaincodeAction.Unmarshal(proposalResponsePayload.Extension); err != nil {
		log.Logger.Errorf("Make proposal to transaction fail because parse ChaincodeAction err when check write set: %s", err)
		return true, err
	}
	txReadWriteSet := &wrset.TxReadWriteSet{}
	if err = txReadWriteSet.Unmarshal(chaincodeAction.Results); err != nil {
		log.Logger.Errorf("Make proposal to transaction fail because parse TxReadWriteSet err when check write set: %s", err)
		return true, err
	}
	// 检查写集是否为空
	nsRwset := txReadWriteSet.GetNsRwset()
	for _, v := range nsRwset {
		pKVRWSet := &kvrwset.KVRWSet{}
		if err = pKVRWSet.Unmarshal(v.GetRwset()); err != nil {
			log.Logger.Errorf("parse KVRWSet err when check write set: %s", err)
		}
		if pKVRWSet.Writes != nil && len(pKVRWSet.Writes) > 0 {
			return false, nil
		}
	}
	return true, nil
}

// 根据提案数据创建交易数据封装结构
func (d *RaftDistributor) makeEnvelope(request *proposal) (*cb.Envelope, error) {
	// 获取提案消息头
	hdr, err := utils.GetHeader(request.proposal.Header)
	if err != nil {
		return nil, err
	}
	// 获取提案消息头中的扩展结构
	hdrExt, err := utils.GetChaincodeHeaderExtension(hdr)
	if err != nil {
		return nil, err
	}
	// 获取提案Payload
	pPayl, err := utils.GetChaincodeProposalPayload(request.proposal.Payload)
	if err != nil {
		return nil, err
	}
	// 从提案Payload中获取链码输入
	chaincodeInput := &pb.ChaincodeInput{}
	if err = chaincodeInput.Unmarshal(pPayl.Input); err != nil {
		return nil, err
	}
	// 从提案Payload中获取交易数据所需部分
	propPayloadBytes, err := utils.GetBytesProposalPayloadForTx(pPayl, hdrExt.PayloadVisibility)
	if err != nil {
		return nil, err
	}
	// 序列化链码操作Payload
	chaincodeActionPayload := &pb.ChaincodeActionPayload{
		ChaincodeProposalPayload: propPayloadBytes,
		Action:                   &pb.ChaincodeEndorsedAction{ProposalResponsePayload: request.resPayload, Endorsements: request.endorsers}}
	chaincodeActionPayloadBytes, err := utils.GetBytesChaincodeActionPayload(chaincodeActionPayload)
	if err != nil {
		return nil, err
	}
	// 新建交易数据
	transaction := &pb.Transaction{Actions: []*pb.TransactionAction{{Header: hdr.SignatureHeader, Payload: chaincodeActionPayloadBytes}}}
	// 序列化交易数据
	txBytes, err := utils.GetBytesTransaction(transaction)
	if err != nil {
		return nil, err
	}
	// 新建签名头
	shdr, err := d.signer.NewSignatureHeader()
	if err != nil {
		return nil, err
	}
	// 序列化签名头
	shdrBuf, err := utils.GetBytesSignatureHeader(shdr)
	if err != nil {
		return nil, err
	}
	// 新建交易数据封装结构Payload, 并序列化
	paylBytes, err := utils.GetBytesPayload(&cb.Payload{Header: &cb.Header{ChannelHeader: hdr.ChannelHeader, SignatureHeader: shdrBuf}, Data: txBytes})
	if err != nil {
		return nil, err
	}
	// 对payload签名
	signature, err := d.signer.Sign(paylBytes)
	if err != nil {
		return nil, err
	}
	// 创建最终交易数据封装结构
	hashChainUrl := conf.V.AttachServerAddress + "/postAttach"
	newAttach := map[string]string{}
	if len(hashChainUrl) > 10 {
		hasError := false
		// hash 上链
		for key, attach := range request.attachs {
			h := sha256.New()
			h.Write([]byte(attach))
			hashAttach := h.Sum(nil)
			hash := hex.EncodeToString(hashAttach)
			newAttach[key] = hash

			log.Logger.Info("newattach ", " hash:", hash, " key:", key)

			// clob字段上传到大文件服务器
			err := d.uploadClobToServer(hash, attach, hashChainUrl)
			if err != nil {
				hasError = true
				break
			}
		}
		// 上传服务器成功 hash 值覆盖之前的clob字段 否则clob字段还是上链,防止数据丢失
		if !hasError {
			request.attachs = newAttach
		}
	}
	msg := &cb.Envelope{Payload: paylBytes, Signature: signature, Attachs: request.attachs}
	if len(msg.Attachs) < 1 {
		msg.Attachs = nil
	}
	return msg, nil
}

// 提案转交易,然后排序
func (d *RaftDistributor) makeEnvelopeToOrder(request *proposal, chainID, txID string) {
	// 检查写集, 写集为空的交易不落块
	empty, err := d.writeSetIsEmpty(request)
	if err != nil {
		return
	}
	if empty {
		err := fmt.Errorf("Make proposal to envelope fail because write set is empty of tx %s", txID)
		log.Logger.Error(err)
		// 发送背书成功但写集为空拒绝落块的消息
		sucMessage := &cb.RBCMessage{Type: 7, Data: []byte(err.Error()), ChainID: chainID, TxID: txID}
		go d.clientManager.SendToPeers([]string{request.sourcePeer}, sucMessage)
		return
	}
	// 获取提案对应的链
	c, ok := d.chainManager.GetChain(chainID)
	if !ok {
		log.Logger.Errorf("Make proposal to envelope fail because chain %s was not found", chainID)
		return
	}
	// 创建交易数据封装
	envelope, err := d.makeEnvelope(request)
	if err != nil {
		log.Logger.Errorf("Make proposal to envelope fail because make envelope err of tx %s: %s", txID, err)
		return
	}
	// 过滤该交易
	committer, err := c.Filters().Apply(envelope)
	if err != nil {
		log.Logger.Errorf("Make proposal to envelope fail because of filter error: %s", err)
		return
	}
	// 将交易丢入队列等待共识落块
	for i := 0; i < 3; i++ {
		if c.Enqueue(envelope, committer) {
			return // 正常退出
		}
		time.Sleep(time.Second * 3)
	}

	log.Logger.Errorf("Make proposal to envelope fail because of consenter error of tx %s", txID)
	return
}

func (d *RaftDistributor) uploadClobToServer(hash string, clob string, serverAddress string) error {

	httpRequest := &cb.RBCHttpRequest{Method: "POST", Endpoint: serverAddress}
	params := make(map[string]string)
	params["key"] = hash
	params["attach"] = clob
	httpRequest.Params = params
	resp, err := d.attachClient.Reqest(httpRequest)
	if err != nil {
		log.Logger.Error("upload to server err : ", err)
		return err
	}

	var data map[string]string
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return err
	}

	code := data["code"]
	if code != "200" {
		return errors.New(string(resp))
	}

	return nil
}
