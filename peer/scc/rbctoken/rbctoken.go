package rbctoken

import (
	"crypto/md5"
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
	QueryRBCCName       	string = "queryRBCCName"
	QueryStateHistory       string = "queryStateHistory"
	QueryStateRList       	string = "queryStateRList"
	SetContract       		string = "setContract"
	GetContract       		string = "getContract"
	GetBalance       		string = "getBalance"
	ContractTransfer       	string = "contractTransfer"
	GetTokenList       		string = "getTokenList"
	GetDealList       		string = "getDealList"
	IncreaseSupply       	string = "increaseSupply"
	Mining       			string = "mining"
)

type RBCToken struct {
}

type RBC20 interface {
	Transfer(from string, to string, value int64) (bool, error)
	getBalance(addr string) (int64, error)
}

type ContractEntity struct {
	TxId              string `json:"txId"`              //交易Id
	TxTime            string `json:"txTime"`            //交易时间
	IdKey             string `json:"idKey"`             //对象存储的key
	TokenType         string `json:"tokenType"`         //合约类型
	Owner             string `json:"owner"`             //合约的拥有者 0 普通合约 1 挖矿合约
	Name              string `json:"name"`              //合约名称
	Symbol            string `json:"symbol"`            //合约简称
	TotalSupply       string `json:"totalSupply"`       //总的发行量
	CanIncreaseSupply string `json:"canIncreaseSupply"` //能否增发 0 不可以增发 1 可以增发
	IncreaseCount     string `json:"increaseCount"`     //增发次数
	//BalanceOf		map[string]string	`json:"balanceOf"`			//用户的余额 key 用户id value 余额
}

type DealEntity struct {
	TxId     string `json:"txId"`     //交易Id
	TxTime   string `json:"txTime"`   //交易时间
	IdKey    string `json:"idKey"`    //对象存储的key
	DealId   string `json:"dealId"`   //交易Id号
	DealType string `json:"dealType"` //交易类型 // 0 创建合约 1 转账 （2 为转出 3 为转进）
	Data     string `json:"data"`     //交易数据
}

func (t *RBCToken) Init(_ shim.ChaincodeStubInterface) pb.Response {

	return shim.Success(nil)
}

func (t *RBCToken) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 2 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
	}

	f := string(args[1])
	switch f {
	case QueryRBCCName:
		return shim.Success([]byte("rbctoken"))
	case QueryStateHistory:
		if len(args) <= 0 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}
		return shim.Success(stub.QueryStateHistory(string(args[2])))
	case QueryStateRList:
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}
		return t.queryStateRList(stub, string(args[2]), string(args[3]))
	case SetContract:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		customerEntity, err := t.getCustomerEntity(stub)
		if err != nil {
			return shim.Error(err.Error())
		}
		if customerEntity == nil || customerEntity.CustomerType != "1" { //非超级会员不能修改模型
			return shim.Error("setContract must from super customer")
		}

		return t.SetContract(stub, args[2])
	case GetContract:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.GetContract(stub, string(args[2]))
	case GetBalance:
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.getBalance(stub, string(args[2]), string(args[3]))
	case ContractTransfer:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.tokenTransfer(stub, string(args[2]))
	case GetTokenList:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.getTokenList(stub, string(args[2]))
	case GetDealList:
		if len(args) < 5 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting 4"))
		}

		return t.getDealList(stub, string(args[2]), string(args[3]), string(args[4]))
	case IncreaseSupply:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.increaseSupply(stub, string(args[2]))
	case Mining:
		if len(args) < 5 {
			return shim.Error("Incorrect number of arguments Expecting 5")
		}

		return t.mining(stub, string(args[2]), string(args[3]), string(args[4]))
	default:
		jsonResp := fmt.Sprintf("function %s is not found", f)
		return shim.Error(jsonResp)
	}
}

func (t *RBCToken) queryStateRList(stub shim.ChaincodeStubInterface, rListName string, params string) pb.Response {
	var m map[string]string
	err := jsoniter.Unmarshal([]byte(params), &m)
	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	cpnoStr := m["cpno"]
	cpno := 0
	if len(cpnoStr) > 0 {
		cpno, _ = strconv.Atoi(cpnoStr)
	}

	if cpno < 0 {
		cpno = 0
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
	// 经典的循环条件初始化/条件判断/循环后条件变化
	for i := bNo; i < eNo; i++ {
		customerId := stub.Get(rListName, nSize-i-1)

		height := stub.GetHeightById(rListName, customerId)
		existCustomer := &rbccustomer.CustomerEntity{CustomerId: customerId, CustomerNo: strconv.Itoa(height)}

		pageList.List = append(pageList.List, existCustomer)
	}

	pageListBuf, err := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}

func (t *RBCToken) GetContract(stub shim.ChaincodeStubInterface, contractName string) pb.Response {
	key := "__RBCTOKEN_" + contractName
	contractBuf, err := stub.GetState(key)
	if err != nil || len(contractBuf) == 0 {
		return shim.Error(fmt.Sprintf("contract with key : %s contractBuf : %s  desn't exist : %s ", key, string(contractBuf), err))
	}

	return shim.Success(contractBuf)
}

func (t *RBCToken) SetContract(stub shim.ChaincodeStubInterface, contract []byte) pb.Response {
	rbcContract := &ContractEntity{}
	err := jsoniter.Unmarshal(contract, rbcContract)
	if err != nil {
		return shim.Error(fmt.Sprintf("unmarshal contract err : %s", err))
	}

	if len(rbcContract.Name) < 1 {
		return shim.Error("contract name is empty")
	}

	key := "__RBCTOKEN_" + rbcContract.Name

	rbcContract.IdKey = key
	rbcContract.TxId = stub.GetTxID()
	customer, err := t.getCustomerEntity(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	rbcContract.Owner = customer.CustomerId
	rbcContract.IncreaseCount = "0"
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	rbcContract.TxTime = txTimeStr

	contractBuf, err := jsoniter.Marshal(rbcContract)
	if err != nil {
		return shim.Error(fmt.Sprintf("unmarshal rbcContract err : %s", err))
	}

	//custom := t.getCustomerEntity(stub)
	ownerKey := "__RCOUNTER_" + "__RBCTOKEN_" + rbcContract.Name + "_" + rbcContract.Owner

	stub.PutState(key, contractBuf)

	float64Val, err := strconv.ParseFloat(rbcContract.TotalSupply, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("strconv.Atoi err : %s", err))
	}
	balance, err := jsoniter.Marshal(float64Val)
	if err != nil {
		return shim.Error(fmt.Sprintf("Marshal contract err : %s", err))
	}
	stub.PutState(ownerKey, balance)

	stub.Add("__RBCTOKEN_", key)
	tokenName := "__RBCTOKEN_" + rbcContract.TokenType
	stub.Add(tokenName, key)

	debugStr := rbcContract.Name + " with key : " + key + " with value: %s " + string(contractBuf) + " deployed success!"

	transDel := &DealEntity{}
	transDel.TxId = stub.GetTxID()
	txTime, _ = stub.GetTxTimestamp()
	tm = time.Unix(txTime.Seconds, 0)
	txTimeStr = tm.Format("2006-01-02 15:04:05")
	transDel.TxTime = txTimeStr

	m := md5.New()
	m.Write([]byte(txTimeStr + string(contract)))
	md5Str := hex.EncodeToString(m.Sum(nil))

	transDel.IdKey = md5Str
	transDel.DealId = md5Str
	transDel.Data = string(contract)
	transDel.DealType = "0"

	transkey := "__RBCTransaction_" + md5Str

	transBytes, err := jsoniter.Marshal(transDel)
	if err != nil {
		return shim.Error(fmt.Sprintf("jsoniter.Marshal(transDel) err : %s", err))
	}
	stub.PutState(transkey, transBytes)

	// 增加索引
	stub.Add("__RBCTransaction_", transkey)
	stub.Add("__RBCTransaction_"+transDel.DealType, transkey)
	stub.Add("__RBCTransaction_"+rbcContract.Owner, transkey)
	stub.Add("__RBCTransaction_"+rbcContract.Owner+"_"+transDel.DealType, transkey)

	return shim.Success([]byte(debugStr))
}

func (t *RBCToken) getBalance(stub shim.ChaincodeStubInterface, contractName, addr string) pb.Response {
	Key := "__RCOUNTER_" + "__RBCTOKEN_" + contractName + "_" + addr

	balanceBytes, err := stub.GetState(Key)
	if err != nil {
		return shim.Error(fmt.Sprintf("stub.GetState err : %s \n", err))
	}

	return shim.Success(balanceBytes)
}

func (t *RBCToken) tokenTransfer(stub shim.ChaincodeStubInterface, params string) pb.Response {

	paramMap := map[string]string{}
	err := jsoniter.Unmarshal([]byte(params), &paramMap)
	if err != nil {
		return shim.Error(fmt.Sprintf("unmarshal err : %s params : %s \n", err, params))
	}

	contractName := paramMap["name"]

	from := paramMap["from"]

	customer, err := t.getCustomerEntity(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if customer == nil {
		log.Logger.Infof("getCustomer error : %s", customer)
		return shim.Error(fmt.Sprintf("getCustumer error"))
	}

	if customer.CustomerStatus != "1" {
		return shim.Error(fmt.Sprint("customer status is frozen or unregisted"))
	}

	if from == "" {
		from = customer.CustomerId
	}

	if from != customer.CustomerId {
		return shim.Error(fmt.Sprintf("you are not the owner of the account can not transfer money."))
	}
	to := paramMap["to"]
	value := paramMap["value"]

	key := "__RBCTOKEN_" + contractName
	contractBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("GetState err : %s", err))
	}

	contractEnt := &ContractEntity{}
	err = jsoniter.Unmarshal(contractBytes, contractEnt)
	if err != nil {
		return shim.Error(fmt.Sprintf("unmarshal contract err : %s", err))
	}

	intVal, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("strconv.Atoi err : %s", err))
	}

	fromKey := "__RCOUNTER_" + "__RBCTOKEN_" + contractName + "_" + from
	toKey := "__RCOUNTER_" + "__RBCTOKEN_" + contractName + "_" + to

	fromBalValBytes, err := stub.GetState(fromKey)
	toBalValBytes, err := stub.GetState(toKey)

	var fromBalVal, toBalVal int

	jsoniter.Unmarshal(fromBalValBytes, &fromBalVal)
	jsoniter.Unmarshal(toBalValBytes, &toBalVal)

	if intVal < 0 {
		return shim.Error(fmt.Sprintf("value can not be negtive"))
	}

	if int64(fromBalVal) < intVal {
		return shim.Error(fmt.Sprintf("ballance not enough fromBalVal is %d toBalVal is %d \n", fromBalVal, toBalVal))
	}

	fromAddKey := "__RBCTOKEN_" + contractName + "_" + from
	toAddKey := "__RBCTOKEN_" + contractName + "_" + to

	stub.AddValue(fromAddKey, -intVal)
	stub.AddValue(toAddKey, intVal)

	transDel := &DealEntity{}
	transDel.TxId = stub.GetTxID()
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	transDel.TxTime = txTimeStr

	m := md5.New()
	m.Write([]byte(txTimeStr + params))
	md5Str := hex.EncodeToString(m.Sum(nil))

	transDel.IdKey = md5Str
	transDel.DealId = md5Str
	transDel.Data = params
	transDel.DealType = "1"

	transkey := "__RBCTransaction_" + md5Str
	transBytes, err := jsoniter.Marshal(transDel)
	if err != nil {
		return shim.Error(fmt.Sprintf("jsoniter.Marshal(transDel) err : %s", err))
	}

	stub.PutState(transkey, transBytes)

	// 增加索引
	stub.Add("__RBCTransaction_", transkey)
	stub.Add("__RBCTransaction_"+transDel.DealType, transkey)
	stub.Add("__RBCTransaction_"+from, transkey)
	stub.Add("__RBCTransaction_"+from+"_2", transkey)
	stub.Add("__RBCTransaction_"+from+"_"+transDel.DealType, transkey)
	stub.Add("__RBCTransaction_"+to, transkey)
	stub.Add("__RBCTransaction_"+to+"_3", transkey)
	stub.Add("__RBCTransaction_"+to+"_"+transDel.DealType, transkey)

	return shim.Success([]byte("contract transfer success"))
}

func (t *RBCToken) getTokenList(stub shim.ChaincodeStubInterface, params string) pb.Response {
	var m map[string]string
	err := jsoniter.Unmarshal([]byte(params), &m)
	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	cpnoStr := m["cpno"]
	cpno := 0
	if len(cpnoStr) > 0 {
		cpno, _ = strconv.Atoi(cpnoStr)
	}

	if cpno < 0 {
		cpno = 0
	}

	rListName := "__RBCTOKEN_"
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
	// 经典的循环条件初始化/条件判断/循环后条件变化
	for i := bNo; i < eNo; i++ {
		tokenId := stub.Get(rListName, i)

		existDeal := &ContractEntity{}
		existBytes, _ := stub.GetState(tokenId)

		err = jsoniter.Unmarshal(existBytes, existDeal)

		if err != nil {
			return shim.Error(fmt.Sprintf("json unmarshal is err: %s", err))
		}

		pageList.List = append(pageList.List, existDeal)
	}

	pageListBuf, err := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}

func (t *RBCToken) getDealList(stub shim.ChaincodeStubInterface, customerid, dealtype, params string) pb.Response {

	var m map[string]string
	err := jsoniter.Unmarshal([]byte(params), &m)
	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	cpnoStr := m["cpno"]
	cpno := 0
	if len(cpnoStr) > 0 {
		cpno, _ = strconv.Atoi(cpnoStr)
	}

	if cpno < 0 {
		cpno = 0
	}

	// dealType = -1 为查询所有交易
	var rListName string
	rListName = "__RBCTransaction_" + customerid + "_" + dealtype
	if dealtype == "-1" {
		rListName = "__RBCTransaction_" + customerid
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
	// 经典的循环条件初始化/条件判断/循环后条件变化
	for i := bNo; i < eNo; i++ {
		tokenId := stub.Get(rListName, i)

		existToken := &DealEntity{}
		existBytes, _ := stub.GetState(tokenId)

		err = jsoniter.Unmarshal(existBytes, existToken)

		if err != nil {
			return shim.Error(fmt.Sprintf("json unmarshal is err: %s", err))
		}

		pageList.List = append(pageList.List, existToken)
	}

	pageListBuf, err := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}

func (t *RBCToken) getCustomerEntity(stub shim.ChaincodeStubInterface) (*rbccustomer.CustomerEntity, error) {

	serializedIdentity := &msp.SerializedIdentity{}

	createBytes, err := stub.GetCreator()
	if err := serializedIdentity.Unmarshal(createBytes); err != nil {
		return nil, err
	}

	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strings.TrimSpace(string(serializedIdentity.IdBytes))))
	md5Str := hex.EncodeToString(md5Ctx.Sum(nil))

	//从install中获取chaincode的内容
	invokeArgs := []string{"query", "queryOne", md5Str}

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
	if err := jsoniter.Unmarshal(resp.Payload, customerEntity); err != nil {
		log.Logger.Errorf("jsoniter.Unmarshal err %s custEnti : %s", err, string(resp.Payload))
		return nil, err
	}

	return customerEntity, nil
}

func (t *RBCToken) increaseSupply(stub shim.ChaincodeStubInterface, tokenName string) pb.Response {

	key := "__RBCTOKEN_" + tokenName
	contractEntiBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("get state err: %s", err))
	}

	contractEntity := &ContractEntity{}
	err = jsoniter.Unmarshal(contractEntiBytes, contractEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("json unmarshal err: %s", err))
	}

	if contractEntity.CanIncreaseSupply == "0" {
		return shim.Error(fmt.Sprintf("this token can not increase supply"))
	}

	customer, err := t.getCustomerEntity(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if customer.CustomerType != "1" {
		return shim.Error(fmt.Sprintf("not super user can not increase supply"))
	}

	tokenBalanceKey := "__RBCTOKEN_" + tokenName + "_" + contractEntity.Owner
	tokenBalBytes, err := stub.GetState(tokenBalanceKey)
	if err != nil {
		return shim.Error(fmt.Sprintf(" stub.GetState(tokenBalanceKey) err: %s", err))
	}

	var tokenBal float64
	err = jsoniter.Unmarshal(tokenBalBytes, &tokenBal)
	if err != nil {
		return shim.Error(fmt.Sprintf(" jsoniter.Unmarshal err: %s", err))
	}

	tolSupFloat, err := strconv.ParseFloat(contractEntity.TotalSupply, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf(" strconv.Atoi(contractEntity.TotalSupply) err: %s", err))
	}

	increaseLine := 0.3 * tolSupFloat
	if tokenBal >= increaseLine {
		return shim.Error(fmt.Sprintf(" balance > 30%% can not increase"))
	}

	increaeVal := 0.2 * tolSupFloat
	tokenBal += increaeVal

	if contractEntity.IncreaseCount == "" {
		contractEntity.IncreaseCount = "0"
	}

	intIncreaseCount, err := strconv.Atoi(contractEntity.IncreaseCount)
	if err != nil {
		return shim.Error(fmt.Sprintf(" strconv.Atoi(contractEntity.IncreaseCount) err : %s", err))
	}

	intIncreaseCount++
	contractEntity.IncreaseCount = strconv.Itoa(intIncreaseCount)
	contractEntity.TotalSupply = strconv.FormatFloat(tolSupFloat+increaeVal, 'E', -1, 64)

	contractBytes, err := jsoniter.Marshal(contractEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("jsoniter.Marshal(contractEntity) err : %s", err))
	}

	log.Logger.Infof("contractEntity is : %s", string(contractBytes))

	stub.PutState(key, contractBytes)
	increaseBytes, err := jsoniter.Marshal(tokenBal)

	if err != nil {
		return shim.Error(fmt.Sprintf(" jsoniter.Marshal(increaeVal) err : %s", err))
	}

	err = stub.PutState(tokenBalanceKey, increaseBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf(" jsoniter.Marshal(increaeVal) err : %s", err))
	}

	txid, err := jsoniter.Marshal(stub.GetTxID())
	if err != nil {
		return shim.Error(fmt.Sprintf(" jsoniter.Marshal(stub.GetTxID()) err : %s", err))
	}

	stub.Add("__RBCTransaction_increase_", tokenBalanceKey)
	stub.Add("__RBCTransaction_increase_"+tokenName, tokenBalanceKey)
	stub.Add("__RBCTransaction_increase_"+tokenName+"_"+contractEntity.Owner, tokenBalanceKey)

	return shim.Success(txid)
}

func (t *RBCToken) mining(stub shim.ChaincodeStubInterface, tokenName, customerId, value string) pb.Response {
	tokenKey := "__RBCTOKEN_" + tokenName

	tokenBytes, err := stub.GetState(tokenKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("stub.GetState(tokenKey) err : %s", err))
	}

	var tokenEntity ContractEntity
	err = jsoniter.Unmarshal(tokenBytes, &tokenEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("jsoniter.Unmarsha err : %s", err))
	}

	if tokenEntity.TokenType != "1" {
		return shim.Error("this token can not mining")
	}

	chr, err := stub.GetChannelHeader()
	if err != nil {
		return shim.Error(fmt.Sprintf("get channel header err : %s", err))
	}

	chaincodeSpec := &pb.ChaincodeSpec{}
	if err = chaincodeSpec.Unmarshal(chr.Extension); err != nil {
		return shim.Error("get chaincode spec has err")
	}

	//如果这些动作不是从elchaincode过来
	if chaincodeSpec.ChaincodeId.Name != "elchaincode" {
		return shim.Error("mining must from elchaincode")
	}

	customerBalanceKey := "__RBCTOKEN_" + tokenName + "_" + customerId

	var tempBalanceInt64 int64
	tempBalanceInt64 = 0

	balanceByte, err := stub.GetState(customerBalanceKey)
	if err != nil {
		if err.Error() == "leveldb: not found" {
			// 初始增发以后不再增发
			log.Logger.Errorf("not find key : %s\n", tokenKey)
			tempBalanceInt64 = 1000
		} else {
			return shim.Error(fmt.Sprintf("get balance err : %s", err))
		}
	}

	var balanceInt64 int64
	if tempBalanceInt64 > 0 {
		balanceInt64 = tempBalanceInt64
	} else {
		err = jsoniter.Unmarshal(balanceByte, &balanceInt64)
		if err != nil {
			return shim.Error(fmt.Sprintf("jsoniter.Unmarshal(balanceByte) err : %s", err))
		}
	}

	valueInt64, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("strconv.ParseInt(value) err : %s", err))
	}

	calBalance := balanceInt64 + valueInt64
	if calBalance < 0 {
		return shim.Error("balance not enough")
	}

	calBalanceByte, err := jsoniter.Marshal(calBalance)
	if err != nil {
		return shim.Error(fmt.Sprintf("jsoniter.Marshal(calBalance) err : %s", err))
	}

	stub.PutState(customerBalanceKey, calBalanceByte)

	stub.Add("__RBCTransaction_mining_", customerBalanceKey)
	stub.Add("__RBCTransaction_mining_"+tokenName, customerBalanceKey)
	stub.Add("__RBCTransaction_mining_"+tokenName+"_"+customerId, customerBalanceKey)

	return shim.Success(calBalanceByte)
}
