package rbccustomer

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/protos/msp"
	pb "github.com/rongzer/blockchain/protos/peer"
)

const (
	queryRBCCName       	string = "queryRBCCName"
	queryStateHistory       string = "queryStateHistory"
	queryStateRList       	string = "queryStateRList"
	register       			string = "register"
	modify       			string = "modify"
	modifyDict       		string = "modifyDict"
	queryOne       			string = "queryOne"
	queryAll       			string = "queryAll"
	testPerform       		string = "testPerform"
	queryPerform       		string = "queryPerform"
	queryPerformByHeight    string = "queryPerformByHeight"
	setRole       			string = "setRole"
	deleteRole       		string = "deleteRole"
	getRole       			string = "getRole"
	getAllRole       		string = "getAllRole"
)

//缓存模型配置信息进内存，不频繁读db
var customerData = struct {
	sync.RWMutex // gard m
	mapBuf       map[string][]byte
}{mapBuf: make(map[string][]byte)}

// rbccustomer Chaincode implementation
type RBCCustomer struct {
}

// Customer data entity
type CustomerEntity struct {
	TxId             string            `json:"txId"`             //交易Id
	TxTime           string            `json:"txTime"`           //交易时间
	IdKey            string            `json:"idKey"`            //对象存储的key
	CustomerId       string            `json:"customerId"`       //会员CustomerId(主键)
	CustomerNo       string            `json:"customerNo"`       //会员编号
	CustomerName     string            `json:"customerName"`     //会员名称
	CustomerType     string            `json:"customerType"`     //会员类型1:超级用户；2：审计用户;3：B端用户；4、C端用户,客户类型不能修改
	CustomerStatus   string            `json:"customerStatus"`   //会员状态:1正常；2锁定；3注销
	CustomerSignCert string            `json:"customerSignCert"` //会员公钥，用于会员验证
	CustomerAuth     string            `json:"customerAuth"`     //会员权限
	RegCustomerNo    string            `json:"regCustomerNo"`    //注册者
	RegTime          string            `json:"regTime"`          //注册时间
	Dict             map[string]string `json:"dict"`             //会员扩展信息Dict
	RoleNos          string            `json:"roleNos"`          //角色信息
	ToCustomerURL    string            `json:"toCustomerURL"`    //通知会员地址
	PeerIDs          string            `json:"peerIDs"`          //节点列表
}

// Role data entity
type RoleEntity struct {
	TxId     string `json:"txId"`     //交易ID
	TxTime   string `json:"txTime"`   //交易时间
	IdKey    string `json:"idKey"`    //对象存储的key
	RoleNo   string `json:"roleNo"`   //角色表主键
	RoleName string `json:"roleName"` //角色名称
	RoleDesc string `json:"roleDesc"` //角色描述

}

// Init initializes the sample system chaincode by storing the key and value
// arguments passed in as parameters
func (t *RBCCustomer) Init(_ shim.ChaincodeStubInterface) pb.Response {
	//as system chaincodes do not take part in consensus and are part of the system,
	//best practice to do nothing (or very little) in Init.

	return shim.Success(nil)
}

// Invoke gets the supplied key and if it exists, updates the key with the newly
// supplied value.
func (t *RBCCustomer) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 2 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
	}

	f := string(args[1])

	switch f {
	case queryRBCCName:
		return shim.Success([]byte("rbccustomer"))
	case queryStateHistory:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		return shim.Success(stub.QueryStateHistory(string(args[2])))
	case queryStateRList:
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments%v. Expecting %d", args, len(args)))
		}
		return t.queryStateRList(stub, string(args[2]), string(args[3]))
	case register:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		return t.register(stub, string(args[2]))
	case modify:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		customerEntity, existCustomer, err := t.getCustomerInfo(stub, string(args[2]))
		if err != nil {
			resp := fmt.Sprintf("modify customer err %s", err)
			shim.Error(resp)
		}
		return t.modifyInfo(stub, customerEntity, existCustomer)
	case modifyDict:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		customerEntity, existCustomer, err := t.getCustomerInfo(stub, string(args[2]))
		if err != nil {
			resp := fmt.Sprintf("modify customer err %s", err)
			shim.Error(resp)
		}
		customerEntity.CustomerName = ""
		customerEntity.CustomerType = ""
		customerEntity.CustomerStatus = ""
		customerEntity.CustomerSignCert = ""
		customerEntity.CustomerAuth = ""

		return t.modifyInfo(stub, customerEntity, existCustomer)
	case queryOne:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		return t.queryOne(stub, string(args[2]))
	case queryAll:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		return t.queryAll(stub, string(args[2]))
	case testPerform:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		stub.PutState(string(args[2]), args[2])
		if strings.HasPrefix(string(args[2]), "RLIST") {
			stub.Add("testPerform", string(args[2]))
		}
		return shim.Success(nil)
	case queryPerform:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		out, err := exec.Command("docker-compose", "-f", "/opt/gopath/src/github.com/rongzer/blockchain/sdkintegration/docker-compose-peer.yaml docker-compose-peer.yaml", "up", "--force-recreate", "-d").Output()
		if err != nil {
			log.Logger.Errorf("\nexec command err:%s\n", err)
		} else {
			log.Logger.Debugf("\nexec command result:%s\n", string(out))
		}

		return t.queryPerform(stub, string(args[2]))
	case queryPerformByHeight:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		return t.queryPerformByHeight(stub, string(args[2]))
	case setRole:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		return t.setRole(stub, string(args[2]))
	case deleteRole:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}

		return t.deleteRole(stub, string(args[2]))
	case getRole:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting %d", len(args)))
		}
		return t.getRole(stub, string(args[2]))
	case getAllRole:
		return t.getAllRole(stub)
	default:
		jsonResp := fmt.Sprintf("function %s is not found", f)
		return shim.Error(jsonResp)
	}
}

// 查询交易访问者信息
func (t *RBCCustomer) getTxCustomerEntity(stub shim.ChaincodeStubInterface) (*CustomerEntity, error) {
	// 取当前用户
	serializedIdentity := &msp.SerializedIdentity{}
	createBytes, _ := stub.GetCreator()
	if err := serializedIdentity.Unmarshal(createBytes); err != nil {
		return nil, err
	}

	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strings.TrimSpace(string(serializedIdentity.IdBytes))))
	md5Str := hex.EncodeToString(md5Ctx.Sum(nil))

	customerBuf, err := stub.GetState("__RBC_IDX_" + md5Str)
	if err != nil {
		return nil, err
	}
	customerKey := string(customerBuf)
	if len(customerKey) < 1 {
		return nil, errors.New("get empty customer key")
	}
	existBytes, err := stub.GetState(customerKey)
	if err != nil {
		return nil, err
	}

	customerEntity := &CustomerEntity{}
	if err = jsoniter.Unmarshal(existBytes, customerEntity); err != nil {
		return nil, err
	}
	return customerEntity, nil
}

// register customer
// arguments passed in as parameters
func (t *RBCCustomer) register(stub shim.ChaincodeStubInterface, params string) pb.Response {
	//获取注册者身份
	//查询原有多少会员
	nCustomerSize := stub.Size("customer")
	customerEntity := &CustomerEntity{}

	if err := jsoniter.Unmarshal([]byte(params), customerEntity); err != nil {
		return shim.Error(fmt.Sprintf("register param is not a json, %s", err))
	}

	//存在会员,就需要进行交易者身份判断
	if nCustomerSize > 0 {
		//获取注册者
		txCustomerEntity, err := t.getTxCustomerEntity(stub)
		if err != nil {
			return shim.Error(err.Error())
		}
		customerType, err := strconv.Atoi(customerEntity.CustomerType)
		if err != nil {
			return shim.Error(fmt.Sprintf("invalid customer type: %s", customerEntity.CustomerType))
		}
		txCustomerType, err := strconv.Atoi(txCustomerEntity.CustomerType)
		if err != nil {
			return shim.Error(fmt.Sprintf("invalid tx customer type: %s", txCustomerEntity.CustomerType))
		}
		//只有1，2，3类型才能注册用户
		if txCustomerType > 3 || txCustomerType < 1 {
			return shim.Error(fmt.Sprintf("tx user customer type is %s not in [1,2,3]", txCustomerEntity.CustomerType))
		}

		//1,2,3类型只能由1用户注册
		if customerType < 4 && customerType != 3 && txCustomerType != 1 {
			return shim.Error(fmt.Sprintf("customer type %s must regist by 1", customerEntity.CustomerType))
		}

		if customerType == 3 && (txCustomerType == 2 || txCustomerType > 3) {
			return shim.Error(fmt.Sprintf("customer type %s must regist by 1 or 3", customerEntity.CustomerType))
		}
		customerEntity.RegCustomerNo = txCustomerEntity.CustomerNo
	} else { //不存在会员,允许直接注册
		customerEntity.RegCustomerNo = "system"
	}

	//默认会员状态是正常
	customerEntity.CustomerStatus = "1"
	if customerEntity.CustomerId == "" || len(customerEntity.CustomerId) < 1 {
		return shim.Error(fmt.Sprintf("customerId is nil"))
	}

	//检查会员是否己经存在
	existBytes, err := stub.GetState("__RBC_" + customerEntity.CustomerId)
	if existBytes != nil {
		return shim.Error(fmt.Sprintf("the customerId %s has exist", customerEntity.CustomerId))
	}

	existBytes, err = stub.GetState("__RBC_IDX_" + customerEntity.CustomerNo)
	if existBytes != nil {
		return shim.Error(fmt.Sprintf("the customerNo %s has exist", customerEntity.CustomerNo))
	}

	customerEntity.IdKey = customerEntity.CustomerId
	customerEntity.TxId = stub.GetTxID()
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	customerEntity.TxTime = txTimeStr
	customerEntity.RegTime = txTimeStr

	//保存会员信息bool
	customerBytes, err := jsoniter.Marshal(customerEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("convert customer entity to json err, %s", err))
	}

	customerKey := "__RBC_" + customerEntity.CustomerId

	stub.PutState(customerKey, customerBytes)
	//维护索引
	stub.PutState("__RBC_IDX_"+customerEntity.CustomerId, []byte(customerKey))
	stub.PutState("__RBC_IDX_"+customerEntity.CustomerNo, []byte(customerKey))

	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strings.TrimSpace(customerEntity.CustomerSignCert)))
	md5Str := hex.EncodeToString(md5Ctx.Sum(nil))
	stub.PutState("__RBC_IDX_"+md5Str, []byte(customerKey))

	//维护列表
	stub.Add("customer", customerKey)
	stub.Add("customer_"+customerEntity.CustomerType, customerKey)
	stub.Add("customer_"+customerEntity.CustomerType+"_"+customerEntity.CustomerStatus, customerKey)
	stub.Add("customer__"+customerEntity.CustomerStatus, customerKey)

	return shim.Success([]byte("regist success"))
}

// arguments passed in as parameters
func (t *RBCCustomer) getCustomerInfo(stub shim.ChaincodeStubInterface, params string) (*CustomerEntity, *CustomerEntity, error) {
	customerEntity := &CustomerEntity{}

	err := jsoniter.Unmarshal([]byte(params), customerEntity)
	if err != nil {
		return nil, nil, fmt.Errorf("register param is not a json, %s", err)
	}
	customerKey := "__RBC_" + customerEntity.CustomerId

	//检查会员是否己经存在
	existBytes, err := stub.GetState(customerKey)
	if existBytes == nil {
		return nil, nil, fmt.Errorf("the customerId %s not exist", customerEntity.CustomerId)
	}

	existCustomer := &CustomerEntity{}
	err = jsoniter.Unmarshal(existBytes, existCustomer)
	if err != nil {
		return nil, nil, fmt.Errorf("exist customer is not a json, %s", err)
	}

	return customerEntity, existCustomer, nil
}

// arguments passed in as parameters
func (t *RBCCustomer) modifyInfo(stub shim.ChaincodeStubInterface, customerEntity *CustomerEntity, existCustomer *CustomerEntity) pb.Response {

	//获取交易发起者
	if _, err := t.getTxCustomerEntity(stub); err != nil {
		return shim.Error(err.Error())
	}

	//修改会员名称、会员类型、会员状态、认证状态、会员证书、角色类型必须由联合审批进行
	if len(customerEntity.CustomerName) > 0 || len(customerEntity.CustomerType) > 0 || len(customerEntity.CustomerStatus) > 0 || len(customerEntity.CustomerAuth) > 0 || len(customerEntity.CustomerSignCert) > 0 || len(customerEntity.RoleNos) > 0 {
		//修改固由参数必须由超级用户进行
		//if txCustomerEntity.CustomerType != "1" {
		//	return shim.Error("modify customer attribute must be 1")
		//}
		chr, err := stub.GetChannelHeader()
		if err != nil {
			return shim.Error("get channel header has err")
		}
		chaincodeSpec := &pb.ChaincodeSpec{}
		if err := chaincodeSpec.Unmarshal(chr.Extension); err != nil {
			return shim.Error("get chaincode spec has err")
		}

		//如果这些动作不是从rbcapproval过来
		if chaincodeSpec.ChaincodeId.Name != "rbcapproval" {
			return shim.Error("modify customer attribute must from rbcapproval")
		}
	}

	if len(customerEntity.CustomerName) > 0 {
		existCustomer.CustomerName = customerEntity.CustomerName
	}

	if len(customerEntity.CustomerAuth) > 0 {
		existCustomer.CustomerAuth = customerEntity.CustomerAuth
	}
	if len(customerEntity.RoleNos) > 0 {
		existCustomer.RoleNos = customerEntity.RoleNos
	}

	oldCustomerType := existCustomer.CustomerType
	oldCustomerStatus := existCustomer.CustomerStatus

	if len(customerEntity.CustomerType) > 0 {
		existCustomer.CustomerType = customerEntity.CustomerType
	}
	if len(customerEntity.ToCustomerURL) > 0 {
		existCustomer.ToCustomerURL = customerEntity.ToCustomerURL
	}
	if len(customerEntity.PeerIDs) > 0 {
		existCustomer.PeerIDs = customerEntity.PeerIDs
	}
	if len(customerEntity.CustomerStatus) > 0 {
		existCustomer.CustomerStatus = customerEntity.CustomerStatus
	}

	if len(customerEntity.CustomerSignCert) > 0 {
		md5Ctx := md5.New()
		md5Ctx.Write([]byte(strings.TrimSpace(existCustomer.CustomerSignCert)))
		md5Str := hex.EncodeToString(md5Ctx.Sum(nil))

		//删除索引
		stub.DelState("__RBC_IDX_" + md5Str)
		existCustomer.CustomerSignCert = customerEntity.CustomerSignCert
		md5Ctx1 := md5.New()
		md5Ctx1.Write([]byte(strings.TrimSpace(customerEntity.CustomerSignCert)))
		md5Str1 := hex.EncodeToString(md5Ctx1.Sum(nil))

		stub.PutState("__RBC_IDX_"+md5Str1, []byte("__RBC_"+customerEntity.CustomerId))
	}

	customerKey := "__RBC_" + existCustomer.CustomerId

	if oldCustomerType != existCustomer.CustomerType {
		stub.Remove("customer_"+oldCustomerType, customerKey)
		stub.Add("customer_"+existCustomer.CustomerType, customerKey)
		stub.Remove("customer_"+oldCustomerType+"_"+oldCustomerStatus, customerKey)
		stub.Add("customer_"+existCustomer.CustomerType+"_"+existCustomer.CustomerStatus, customerKey)

	} else if oldCustomerStatus != existCustomer.CustomerStatus {
		stub.Remove("customer_"+oldCustomerType+"_"+oldCustomerStatus, customerKey)
		stub.Add("customer_"+existCustomer.CustomerType+"_"+existCustomer.CustomerStatus, customerKey)
		stub.Remove("customer__"+oldCustomerStatus, customerKey)
		stub.Add("customer__"+existCustomer.CustomerStatus, customerKey)
	}

	if customerEntity.Dict != nil {
		existCustomer.Dict = customerEntity.Dict
	}

	existCustomer.IdKey = "__RBC_" + customerEntity.CustomerId
	existCustomer.TxId = stub.GetTxID()
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	existCustomer.TxTime = txTimeStr

	//保存会员信息bool
	customerBytes, err := jsoniter.Marshal(existCustomer)
	if err != nil {
		return shim.Error(fmt.Sprintf("convert customer entity to json err, %s", err))
	}
	stub.PutState(customerKey, customerBytes)

	return shim.Success([]byte("regist success"))
}

func (t *RBCCustomer) queryOne(stub shim.ChaincodeStubInterface, params string) pb.Response {
	customerData.Lock()
	defer customerData.Unlock()
	key := "__RBC_IDX_" + params
	if customerData.mapBuf[key] != nil {
		shim.Success(customerData.mapBuf[key])
	}
	customerBuf, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("customer not found %s", err))
	}
	customerKey := string(customerBuf)
	if len(customerKey) < 1 {
		return shim.Success(nil)
	}
	existBytes, err := stub.GetState(customerKey)
	if existBytes == nil {
		return shim.Success([]byte(""))
	}

	customerData.mapBuf[key] = existBytes
	return shim.Success(existBytes)
}

func (t *RBCCustomer) queryAll(stub shim.ChaincodeStubInterface, params string) pb.Response {
	var m map[string]string
	err := jsoniter.Unmarshal([]byte(params), &m)
	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	// cpno 0:第一页 1：第二页
	cpnoStr := m["cpno"]
	cpno := 0
	if len(cpnoStr) > 0 {
		cpno, _ = strconv.Atoi(cpnoStr)
	}

	if cpno < 0 {
		cpno = 0
	}

	customerType := m["customerType"]
	customerStatus := m["customerStatus"]

	rListName := "customer"
	if len(customerType) > 0 {
		rListName = rListName + "_" + customerType
		if len(customerStatus) > 0 {
			rListName = rListName + "_" + customerStatus
		}
	} else {
		if len(customerStatus) > 0 {
			rListName = rListName + "__" + customerStatus
		}
	}

	nSize := stub.Size(rListName)

	if cpno > nSize/20 {
		cpno = nSize / 20
	}

	bNo := cpno * 20
	eNo := (cpno + 1) * 20

	if eNo > nSize {
		eNo = nSize
	}

	listValue := make([]interface{}, 0)

	pageList := &shim.PageList{Cpno: strconv.Itoa(cpno), Rnum: strconv.Itoa(nSize), List: listValue}
	// 经典的循环条件初始化/条件判断/循环后条件变化
	for i := bNo; i < eNo; i++ {
		customerKey := stub.Get(rListName, i)
		if len(customerKey) > 0 {
			existBytes, _ := stub.GetState(customerKey)
			if existBytes != nil {
				existCustomer := &CustomerEntity{}
				err = jsoniter.Unmarshal(existBytes, existCustomer)
				if err == nil {
					pageList.List = append(pageList.List, existCustomer)
				}
			}
		}
	}
	pageListBuf, err := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}

func (t *RBCCustomer) queryPerform(stub shim.ChaincodeStubInterface, params string) pb.Response {
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
	rListName := "testPerform"
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
		customerKey := stub.Get(rListName, nSize-i-1)

		height := stub.GetHeightById(rListName, customerKey)
		existCustomer := &CustomerEntity{CustomerId: customerKey, CustomerNo: strconv.Itoa(height)}

		pageList.List = append(pageList.List, existCustomer)
	}

	pageListBuf, err := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}

func (t *RBCCustomer) queryPerformByHeight(stub shim.ChaincodeStubInterface, params string) pb.Response {
	var m map[string]string
	err := jsoniter.Unmarshal([]byte(params), &m)
	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	height, _ := strconv.Atoi(m["height"])
	rListName := "testPerform"

	ids, nextHeight := stub.GetIdsByHeight(rListName, height)

	nSize := stub.Size(rListName)

	listValue := make([]interface{}, 0)

	pageList := &shim.PageList{Cpno: strconv.Itoa(nextHeight), Rnum: strconv.Itoa(nSize), List: listValue}
	// 经典的循环条件初始化/条件判断/循环后条件变化
	for _, v := range ids {
		customerKey := v

		height := stub.GetHeightById(rListName, customerKey)
		existCustomer := &CustomerEntity{CustomerId: customerKey, CustomerNo: strconv.Itoa(height)}

		pageList.List = append(pageList.List, existCustomer)
	}

	pageListBuf, err := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)

}

func (t *RBCCustomer) queryStateRList(stub shim.ChaincodeStubInterface, rListName string, params string) pb.Response {
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
		customerKey := stub.Get(rListName, nSize-i-1)

		height := stub.GetHeightById(rListName, customerKey)
		existCustomer := &CustomerEntity{CustomerId: customerKey, CustomerNo: strconv.Itoa(height)}

		pageList.List = append(pageList.List, existCustomer)
	}

	pageListBuf, err := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}

//新增用户角色
func (t *RBCCustomer) setRole(stub shim.ChaincodeStubInterface, params string) pb.Response {
	roleEntity := &RoleEntity{}
	err := jsoniter.Unmarshal([]byte(params), roleEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("createrRoleInfo param is not a json, %s", err))
	}

	key := "__RBC_ROLE_" + roleEntity.RoleNo
	roleEntity.IdKey = key
	roleEntity.TxId = stub.GetTxID()
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	roleEntity.TxTime = txTimeStr
	roleBytes, err := jsoniter.Marshal(roleEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("RoleEntity Marshal 错误"))
	}
	//维护索引
	stub.PutState(key, roleBytes)
	//维护列表
	stub.Add("__RBC_ROLE_", key)
	return shim.Success(roleBytes)
}

// 删除用户角色信息
func (t *RBCCustomer) deleteRole(stub shim.ChaincodeStubInterface, roleNo string) pb.Response {
	key := "__RBC_ROLE_" + roleNo
	stub.DelState(key)
	stub.Remove("__RBC_ROLE_", key)
	return shim.Success([]byte("delete role success"))
}

// 查询单个用户角色信息
func (t *RBCCustomer) getRole(stub shim.ChaincodeStubInterface, roleNo string) pb.Response {
	key := "__RBC_ROLE_" + roleNo
	roleBuf, _ := stub.GetState(key)
	return shim.Success(roleBuf)
}

// 查询所有用户角色信息
func (t *RBCCustomer) getAllRole(stub shim.ChaincodeStubInterface) pb.Response {
	rListName := "__RBC_ROLE_"
	nSize := stub.Size(rListName)

	listValue := make([]interface{}, 0)

	pageList := &shim.PageList{Cpno: strconv.Itoa(0), Rnum: strconv.Itoa(nSize), List: listValue}
	// 循环获取每页表信息
	for i := 0; i < nSize; i++ {

		existRole := &RoleEntity{}
		key := stub.Get(rListName, i)

		if len(key) > 0 {
			existBytes, _ := stub.GetState(key)

			if existBytes != nil {
				err := jsoniter.Unmarshal(existBytes, existRole)
				if err == nil {
					pageList.List = append(pageList.List, existRole)
				}
			}
		}
	}
	pageListBuf, _ := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}

func ClearCache() {
	customerData.Lock()
	defer customerData.Unlock()
	customerData.mapBuf = make(map[string][]byte)
}
