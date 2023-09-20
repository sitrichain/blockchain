package rbcapproval

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/scc/rbccustomer"
	"github.com/rongzer/blockchain/protos/msp"
	pb "github.com/rongzer/blockchain/protos/peer"
)

const (
	queryRBCCName     string = "queryRBCCName"
	queryStateHistory string = "queryStateHistory"
	newApproval       string = "newApproval"
	approval          string = "approval"
	queryOneApproval  string = "queryOneApproval"
	queryAllApproval  string = "queryAllApproval"
	ChainCodeName     string = "rbcapproval"
)

// rbccustomer Chaincode implementation
type RBCApproval struct {
}

//会员签名
type Sign struct {
	TxId         string `json:"txId"`         //交易ID
	TxTime       string `json:"txTime"`       //交易时间
	CustomerNo   string `json:"customerNo"`   //会员No
	Status       string `json:"status"`       //是否同意(1:同意；2:不同意)
	Desc         string `json:"desc"`         //备注
	ApprovalTime string `json:"approvalTime"` //签名时间
}

//审批事项
type ApprovalEntity struct {
	TxId   string `json:"txId"`   //交易ID
	TxTime string `json:"txTime"` //交易时间
	IdKey  string `json:"idKey"`  //对象存储的key

	ApprovalId   string   `json:"approvalId"`   //Id(主键)
	ApprovalName string   `json:"approvalName"` //审批事项
	ApprovalType string   `json:"approvalType"` //审批事项类型，判断执行哪个方法(deployChainCode:版本升级,upgradeChainCode:版本升级,setVerChangeCode:版本变更...)
	VetoType     string   `json:"vetoType"`     //事项投票类型，判断是否执行方法(1:一票否决型,2:2/3以上同意,3:半数以上同意)
	Chaincode    string   `json:"chaincode"`    //执行的智能合约
	Func         string   `json:"func"`         //执行合约的方法
	Args         []string `json:"args"`         //执行方法的参数
	Creater      string   `json:"creater"`      //事项发起人CustomerNo
	CreateTime   string   `json:"createTime"`   //发起时间
	Status       string   `json:"status"`       //事项状态(1:审批中,2:己通过,3:未通过)
	Signs        []*Sign  `json:"signs"`        //会员签名
	Desc         string   `json:"desc"`         //备注
	Dict         string   `json:"dict"`         //扩展属性
}

// Init initializes the sample system chaincode by storing the key and value
// arguments passed in as parameters
func (t *RBCApproval) Init(_ shim.ChaincodeStubInterface) pb.Response {
	//as system chaincodes do not take part in consensus and are part of the system,
	//best practice to do nothing (or very little) in Init.

	return shim.Success(nil)
}

// Invoke gets the supplied key and if it exists, updates the key with the newly
// supplied value.
func (t *RBCApproval) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 3 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments%v. Expecting 1", args))
	}

	f := string(args[1])

	switch f {
	case queryRBCCName:
		return shim.Success([]byte(ChainCodeName))
	case queryStateHistory:
		return shim.Success(stub.QueryStateHistory(string(args[2])))
	case newApproval:
		return t.newApproval(stub, string(args[2]))
	case approval:
		return t.approval(stub, string(args[2]))
	case queryOneApproval:
		return t.queryOneApproval(stub, string(args[2]))
	case queryAllApproval:
		return t.queryAllApproval(stub, string(args[2]))
	default:
		jsonResp := fmt.Sprintf("function %s is not found", f)
		return shim.Error(jsonResp)
	}
}

func (t *RBCApproval) getCustomerEntity(stub shim.ChaincodeStubInterface, params string) (*rbccustomer.CustomerEntity, error) {
	//从install中获取chaincode的内容
	invokeArgs := []string{"query", "queryOne", params}

	chr, err := stub.GetChannelHeader()
	if err != nil {
		return nil, err
	}
	resp := stub.InvokeChaincode("rbccustomer", util.ArrayToChaincodeArgs(invokeArgs), chr.ChannelId)
	if resp.Status != shim.OK {
		log.Logger.Errorf("get customerEntity is err %s", resp.Message)

		return nil, errors.New(resp.Message)
	}
	//将chaincode的代码设置到参数中(ChainCodeData)
	customerEntity := &rbccustomer.CustomerEntity{}
	err = jsoniter.Unmarshal(resp.Payload, customerEntity)
	if err != nil {
		return nil, err
	}

	return customerEntity, nil
}

func (t *RBCApproval) getBCustomerNum(stub shim.ChaincodeStubInterface) int {
	//从install中获取chaincode的内容
	invokeArgs := []string{"query", "queryAll", "{\"cpno\":\"0\",\"customerType\":\"3\",\"customerStatus\":\"1\"}"}

	chr, _ := stub.GetChannelHeader()
	resp := stub.InvokeChaincode("rbccustomer", util.ArrayToChaincodeArgs(invokeArgs), chr.ChannelId)
	if resp.Status != shim.OK {
		log.Logger.Errorf("get SuperCustomerNum is err %s", resp.Message)

		return -1
	}

	//将chaincode的代码设置到参数中(ChainCodeData)
	pageList := &shim.PageList{}
	err := jsoniter.Unmarshal(resp.Payload, pageList)
	if err != nil {
		log.Logger.Errorf("get SuperCustomerNum is err %s", err)
		return -1
	}
	rnum, _ := strconv.Atoi(pageList.Rnum)

	return rnum
}

// Init initializes the sample system chaincode by storing the key and value
// arguments passed in as parameters
func (t *RBCApproval) newApproval(stub shim.ChaincodeStubInterface, params string) pb.Response {
	approvalEntity := &ApprovalEntity{}
	//log.Logger.Infof("new approval param:" + params)
	err := jsoniter.Unmarshal([]byte(params), approvalEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("new approval param %s is not a json, %s", params, err))
	}

	approvalEntity.Signs = make([]*Sign, 0)
	approvalEntity.Status = "1"

	for i := 0; i < len(approvalEntity.Args); i++ {
		approvalEntity.Args[i] = strings.TrimSpace(approvalEntity.Args[i])
	}

	existByte, err := stub.GetState("__RBC_approval_" + approvalEntity.ApprovalId)
	if existByte != nil {
		return shim.Error(fmt.Sprintf("the approval %s has exist", approvalEntity.ApprovalId))

	}
	//取当前用户
	serializedIdentity := &msp.SerializedIdentity{}
	createBytes, _ := stub.GetCreator()
	if err = serializedIdentity.Unmarshal(createBytes); err != nil {
		return shim.Error(fmt.Sprintf("Could not deserialize a SerializedIdentity, err %s", err))
	}
	md5Ctx := md5.New()

	md5Ctx.Write([]byte(strings.TrimSpace(string(serializedIdentity.IdBytes))))
	md5Str := hex.EncodeToString(md5Ctx.Sum(nil))

	customerEntity, err := t.getCustomerEntity(stub, md5Str)
	if err != nil {
		return shim.Error(fmt.Sprintf("get customer err : %s", err.Error()))
	}

	if customerEntity == nil {
		return shim.Error(fmt.Sprintf("new approval customer is not exist"))
	}

	if customerEntity.CustomerStatus != "1" {
		return shim.Error(fmt.Sprintf("tx customer status %s is invalid", customerEntity.CustomerStatus))
	}

	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	approvalEntity.CreateTime = txTimeStr
	approvalEntity.Creater = customerEntity.CustomerNo

	//判断是否是超级用户
	if customerEntity.CustomerType == "1" {
		sign := &Sign{CustomerNo: customerEntity.CustomerNo, TxId: stub.GetTxID(), Status: "1", ApprovalTime: txTimeStr}
		sign.Desc = approvalEntity.Desc
		approvalEntity.Signs = append(approvalEntity.Signs, sign)
		approvalEntity.Status = "2"
	}

	//对版本发布与升级智能合约特殊处理
	if approvalEntity.Chaincode == "lscc" && (approvalEntity.ApprovalType == "deployChainCode" || approvalEntity.ApprovalType == "upgradeChainCode") {
		//从install中获取chaincode的内容
		invokeArgs := []string{"getInstallChainCode"}

		invokeArgs = append(invokeArgs, approvalEntity.Args...)
		chdr, err := stub.GetChannelHeader()
		if err != nil {
			return shim.Error(fmt.Sprintf("get channel header err, %s", err))
		}
		resp := stub.InvokeChaincode("lscc", util.ArrayToChaincodeArgs(invokeArgs), chdr.ChannelId)
		if resp.Status != shim.OK {
			errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", resp.Message)
			return shim.Error(errStr)
		}

		//将chaincode的代码设置到参数中(ChainCodeData)
		newArgs := []string{approvalEntity.Args[0], approvalEntity.Args[1], approvalEntity.Args[2], approvalEntity.Args[3], base64.StdEncoding.EncodeToString(resp.Payload)}
		approvalEntity.Args = newArgs
	}

	/*BCustomerSize := t.getBCustomerNum(stub)

	if BCustomerSize <= len(approvalEntity.Signs) {
		//计算审批通过数
		approvalIn := 0
		approvalOut := 0
		// 经典的循环条件初始化/条件判断/循环后条件变化,倒序
		for i := 0; i < len(approvalEntity.Signs); i++ {
			sign := approvalEntity.Signs[i]
			if sign.Status == "1" {
				approvalIn++
			} else {
				approvalOut++
			}
		}

		if approvalEntity.VetoType == "1" { //1:一票否决型,2:2/3以上同意,3:半数以上同意
			if approvalOut > 0 {
				approvalEntity.Status = "3"
			} else if approvalIn >= BCustomerSize {
				approvalEntity.Status = "2"
			}
		} else if approvalEntity.VetoType == "2" {
			if approvalOut > (BCustomerSize+1)/3 {
				approvalEntity.Status = "3"
			} else if approvalIn >= 2*(BCustomerSize+1)/3 {
				approvalEntity.Status = "2"
			}

		} else if approvalEntity.VetoType == "3" {
			if approvalOut > (BCustomerSize+1)/2 {
				approvalEntity.Status = "3"
			} else if approvalIn > (BCustomerSize+1)/2 {
				approvalEntity.Status = "2"
			}
		}
	}*/

	if approvalEntity.Status == "2" {
		//审批通过，执行合约交易
		invokeArgs := []string{approvalEntity.Func}
		invokeArgs = append(invokeArgs, approvalEntity.Args...)

		chdr, err := stub.GetChannelHeader()
		if err != nil {
			return shim.Error(fmt.Sprintf("get channel header err, %s", err))
		}

		resp := stub.InvokeChaincode(approvalEntity.Chaincode, util.ArrayToChaincodeArgs(invokeArgs), chdr.ChannelId)
		if resp.Status != shim.OK {
			log.Logger.Errorf("Failed to invoke chaincode %s by approval. Got error: %s", approvalEntity.Chaincode, resp.Message)
		} else {
			log.Logger.Infof("success invoke chaincode %s by approval.", approvalEntity.Chaincode)
		}
	}

	approvalEntity.IdKey = "__RBC_approval_" + approvalEntity.ApprovalId
	approvalEntity.TxId = stub.GetTxID()
	approvalEntity.TxTime = txTimeStr

	//保存会员信息bool
	approvalBytes, err := jsoniter.Marshal(approvalEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("convert approval entity to json err, %s", err))
	}

	approvelKey := "__RBC_approval_" + approvalEntity.ApprovalId
	//保存列表
	stub.PutState(approvelKey, approvalBytes)

	//维护列表
	stub.Add("approval", approvelKey)
	stub.Add("approval_"+approvalEntity.Status, approvelKey)

	return shim.Success([]byte("new approval success"))
}

// Init initializes the sample system chaincode by storing the key and value
// arguments passed in as parameters
func (t *RBCApproval) approval(stub shim.ChaincodeStubInterface, params string) pb.Response {
	approvalEntity := &ApprovalEntity{}

	err := jsoniter.Unmarshal([]byte(params), approvalEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("new approval param is not a json, %s", err))
	}

	if approvalEntity.Status != "1" && approvalEntity.Status != "2" {
		return shim.Error(fmt.Sprintf("approval status %s is not invalid", approvalEntity.Status))
	}

	existBytes, err := stub.GetState("__RBC_approval_" + approvalEntity.ApprovalId)
	if err != nil {
		return shim.Error(fmt.Sprintf("approval %s is not exist, %s", approvalEntity.ApprovalId, err))
	}

	existEntity := &ApprovalEntity{}

	err = jsoniter.Unmarshal(existBytes, existEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("approval param is not a json, %s", err))
	}

	if existEntity.Status != "1" { //判断是否在审批中
		return shim.Error(fmt.Sprintf("approval %s has over", existEntity.ApprovalId))
	}

	//取当前用户
	serializedIdentity := &msp.SerializedIdentity{}
	createBytes, _ := stub.GetCreator()
	if err := serializedIdentity.Unmarshal(createBytes); err != nil {
		return shim.Error(fmt.Sprintf("Could not deserialize a SerializedIdentity, err %s", err))
	}
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strings.TrimSpace(string(serializedIdentity.IdBytes))))
	md5Str := hex.EncodeToString(md5Ctx.Sum(nil))

	customerEntity, err := t.getCustomerEntity(stub, md5Str)
	if err != nil ||  customerEntity == nil {
		return shim.Error(fmt.Sprintf("approval customer is not exist : %s", err))
	}

	if customerEntity.CustomerStatus != "1" {
		return shim.Error(fmt.Sprintf("tx customer status %s is invalid", customerEntity.CustomerStatus))
	}

	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")

	//判断用户是否己经审批过
	for i := 0; i < len(existEntity.Signs); i++ {
		sign := existEntity.Signs[i]
		if sign.CustomerNo == customerEntity.CustomerNo {
			return shim.Error(fmt.Sprintf("customer %s have approvaled", customerEntity.CustomerNo))
		}
	}

	////判断是否是超级用户
	if customerEntity.CustomerType == "1" {
		if "1" == approvalEntity.Status {
			// 超级会员一票通过
			existEntity.Status = "2"

			sign := &Sign{CustomerNo: customerEntity.CustomerNo, TxId: stub.GetTxID(), Status: approvalEntity.Status, ApprovalTime: txTimeStr}
			sign.Desc = approvalEntity.Desc
			sign.TxTime = txTimeStr
			existEntity.Signs = append(existEntity.Signs, sign)
		} else {
			// 超级会员一票否决
			existEntity.Status = "3"

			sign := &Sign{CustomerNo: customerEntity.CustomerNo, TxId: stub.GetTxID(), Status: approvalEntity.Status, ApprovalTime: txTimeStr}
			sign.Desc = approvalEntity.Desc
			sign.TxTime = txTimeStr
			existEntity.Signs = append(existEntity.Signs, sign)
		}
	} else {
		// 普通会员需要计算投票数 满足条件才能通过
		sign := &Sign{CustomerNo: customerEntity.CustomerNo, TxId: stub.GetTxID(), Status: approvalEntity.Status, ApprovalTime: txTimeStr}
		sign.Desc = approvalEntity.Desc
		sign.TxTime = txTimeStr
		existEntity.Signs = append(existEntity.Signs, sign)

		BUserSize := t.getBCustomerNum(stub)

		if len(existEntity.Signs) >= 1 {
			//计算审批通过数
			approvalIn := 0
			approvalOut := 0
			// 经典的循环条件初始化/条件判断/循环后条件变化,倒序
			for i := 0; i < len(existEntity.Signs); i++ {
				sign := existEntity.Signs[i]
				if sign.Status == "1" {
					approvalIn++
				} else {
					approvalOut++
				}
			}
			if existEntity.VetoType == "1" { //1:一票否决型,2:2/3以上同意,3:半数以上同意
				if approvalOut > 0 {
					existEntity.Status = "3"
				} else if approvalIn >= BUserSize {
					existEntity.Status = "2"
				}
			} else if existEntity.VetoType == "2" {
				if float32(approvalOut) > float32(BUserSize)/3 {
					existEntity.Status = "3"
				} else if float32(approvalIn) >= 2*float32(BUserSize)/3 {
					existEntity.Status = "2"
				}

			} else if existEntity.VetoType == "3" {
				if float32(approvalOut) > float32(BUserSize)/2 {
					existEntity.Status = "3"
				} else if float32(approvalIn) > float32(BUserSize)/2 {
					existEntity.Status = "2"
				}
			}
		}
	}

	if existEntity.Status != "1" {
		approvelKey := "__RBC_approval_" + existEntity.ApprovalId

		stub.Remove("approval_1", approvelKey)

		//维护列表
		stub.Add("approval_"+existEntity.Status, approvelKey)
	}

	if existEntity.Status == "2" {
		//审批通过，执行合约交易
		invokeArgs := []string{existEntity.Func}
		invokeArgs = append(invokeArgs, existEntity.Args...)

		chdr, err := stub.GetChannelHeader()
		if err != nil {
			return shim.Error(fmt.Sprintf("get channel header err, %s", err))
		}
		resp := stub.InvokeChaincode(existEntity.Chaincode, util.ArrayToChaincodeArgs(invokeArgs), chdr.ChannelId)
		if resp.Status != shim.OK {
			log.Logger.Errorf("Failed to invoke chaincode %s by approval. Got error: %s", approvalEntity.Chaincode, resp.Message)
		} else {
			log.Logger.Info("success invoke chaincode %s by approval.", approvalEntity.Chaincode)

		}
	}

	//保存会员信息bool
	approvalBytes, err := jsoniter.Marshal(existEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("convert customer entity to json err, %s", err))
	}

	//保存列表
	stub.PutState("__RBC_approval_"+existEntity.ApprovalId, approvalBytes)

	return shim.Success([]byte("approval success"))
}

func (t *RBCApproval) queryOneApproval(stub shim.ChaincodeStubInterface, params string) pb.Response {

	existBuf, err := stub.GetState("__RBC_approval_" + params)
	if err != nil {
		return shim.Error(fmt.Sprintf("approval %s found err: %s", params, err))
	}

	return shim.Success(existBuf)
}

func (t *RBCApproval) queryAllApproval(stub shim.ChaincodeStubInterface, params string) pb.Response {
	var m map[string]string
	err := jsoniter.Unmarshal([]byte(params), &m)

	cpnoStr := m["cpno"]
	cpno := 0
	if len(cpnoStr) > 0 {
		cpno, _ = strconv.Atoi(cpnoStr)
	}

	if cpno < 0 {
		cpno = 0
	}

	approvalStatus := m["approvalStatus"]

	rListName := "approval"
	if len(approvalStatus) > 0 {
		rListName = rListName + "_" + approvalStatus
	}

	nSize := stub.Size(rListName)

	if cpno > nSize/100 {
		cpno = nSize / 100
	}

	bNo := cpno * 100
	eNo := (cpno + 1) * 100

	if eNo > nSize {
		eNo = nSize
	}
	listValue := make([]interface{}, 0)

	pageList := &shim.PageList{Cpno: strconv.Itoa(cpno), Rnum: strconv.Itoa(nSize), List: listValue}
	// 经典的循环条件初始化/条件判断/循环后条件变化,倒序
	for i := bNo; i < eNo; i++ {
		approvalKey := stub.Get(rListName, nSize-i-1)
		if len(approvalKey) > 0 {
			existBytes, _ := stub.GetState(approvalKey)
			if existBytes != nil {
				existEntity := &ApprovalEntity{}
				err = jsoniter.Unmarshal(existBytes, existEntity)
				if err == nil {
					pageList.List = append(pageList.List, existEntity)
				}
			}
		}
	}
	pageListBuf, err := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}
