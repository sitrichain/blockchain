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
建模支撑
*/

package rbcmodel

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rongzer/blockchain/peer/broadcastclient"

	jsoniter "github.com/json-iterator/go"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/scc/cache"
	"github.com/rongzer/blockchain/peer/scc/rbccustomer"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/msp"
	"github.com/rongzer/blockchain/protos/orderer"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/spf13/viper"
)

const (
	QueryRBCCName       	string = "queryRBCCName"
	QueryStateHistory       string = "queryStateHistory"
	QueryStateRList       	string = "queryStateRList"
	SetModel       			string = "setModel"
	GetModel       			string = "getModel"
	DeleteModel       		string = "deleteModel"
	GetModelList       		string = "getModelList"
	SetTable       			string = "setTable"
	DeleteTable       		string = "deleteTable"
	GetTable       			string = "getTable"
	GetTableList       		string = "getTableList"
	GetTableListWithPageCat string = "getTableListWithPageCat"
	SetMainPublicKey       	string = "setMainPublicKey"
	GetMainPublicKey       	string = "getMainPublicKey"
	GetMainCryptogram       string = "getMainCryptogram"
	GetFromHttp       		string = "getFromHttp"
	GetAttach       		string = "getAttach"
)

type ModelDataStruct struct {
	//sync.RWMutex // gard m
	modelsCache []*RBCModelInfo
	tablesCache map[string][]*RBCTable
}

//缓存模型配置信息进内存，不频繁读db
//var modelData = ModelDataStruct{mapBuf: make(map[string][]byte)}
//var modelsCache = []*RBCModelInfo{}
//var tablesCache = []*RBCTable{}

var chanModelDatas = struct {
	sync.RWMutex
	mapDataBuf map[string]*ModelDataStruct
}{mapDataBuf: make(map[string]*ModelDataStruct)}

// RBCModel Chaincode implementation
type RBCModel struct {
	cryptoCache *cache.Cache
	signFunc    plugin.Symbol
}

type RBCModelInfo struct {
	TxId           string `json:"txId"`           //交易Id
	TxTime         string `json:"txTime"`         //交易时间
	IdKey          string `json:"idKey"`          //对象存储的key
	ModelName      string `json:"modelName"`      //模型名称
	ModelNameEx    string `json:"modelNameEx"`    //模型别名
	FileRootPath   string `json:"fileRootPath"`   //文件存储根路径
	FileHttpURL    string `json:"fileHttpURL"`    //文件访问URL
	DecryptService string `json:"decryptService"` //解密服务地址
	ModelRoles     string `json:"modelRoles"`     //模型角色权限
	ModelDesc      string `json:"modelDesc"`      //建模备注
	TableAmount    int    `json:"tableAmount"`    //表数量
}

type RBCTable struct {
	TxId           string       `json:"txId"`                //交易Id
	TxTime         string       `json:"txTime"`              //交易时间
	IdKey          string       `json:"idKey"`               //对象存储的key
	TableName      string       `json:"tableName"`           //表名
	TableNameEx    string       `json:"tableNameEx"`         //别名
	TableCategory  string       `json:"tableCategory"`       //表归属分类
	TableCategory1 string       `json:"tableCategory1"`      //表归属分类
	TableCategory2 string       `json:"tableCategory2"`      //表归属分类
	TableType      int          `json:"tableType"`           //表数据类型，1:单条型，2:多条型
	TableExtend    string       `json:"tableExtend"`         //表继承，默认继续自BaseTable
	TableDesc      string       `json:"tableDesc"`           //表备注
	ControlRole    string       `json:"controlRole"`         //控制角色，在此角色下，数据的customerNo必须需访问者的一致方可获取数据密码
	ColList        []*RBCColumn `json:"colList,omitempty"`   //字段列表
	IndexList      []*RBCIndex  `json:"indexList,omitempty"` //索引列表
	ChaincodeNames string       `json:"chaincodeNames"`      //被智能合约引用的数量
	CompensateCon  string       `json:"compensateCon"`       //补偿条件（10h、10d、10M、10Y/-1 不补偿/0 永远补偿）
	CompensateURL  string       `json:"compensateURL"`       //补偿地址
}

type RBCColumn struct {
	TxId          string `json:"txId"`          //交易Id
	TxTime        string `json:"txTime"`        //交易时间
	IdKey         string `json:"idKey"`         //对象存储的key
	ColumnName    string `json:"columnName"`    //字段名
	ColumnNameEx  string `json:"columnNameEx"`  //别名
	ColumnType    string `json:"columnType"`    //字段类型，number:数值,text:文本,date:日期,file:文件，area归属地
	ColumnLength  int    `json:"columnLength"`  //字段长度
	ColumnNotNull int    `json:"columnNotNull"` //非空,默认0:允许为空,1:非空
	ColumnUnique  int    `json:"columnUnique"`  //1:联合唯一键
	ColumnIndex   int    `json:"columnIndex"`   //是否索引,默认0:不索引,1:索引;作查询索引使用
	WriteRoles    string `json:"writeRoles"`    //写权限
	ReadRoles     string `json:"readRoles"`     //读权限
	ColumnDesc    string `json:"columnDesc"`    //字段描述
}

type RBCIndex struct {
	TxId        string `json:"txId"`        //交易Id
	TxTime      string `json:"txTime"`      //交易时间
	IdKey       string `json:"idKey"`       //对象存储的key
	IndexName   string `json:"indexName"`   //索引名称
	IndexNameEx string `json:"indexNameEx"` //别名
	IndexCols   string `json:"indexCols"`   //字段列表
}

type RBCFile struct {
	TxId        string `json:"txId"`        //交易Id
	TxTime      string `json:"txTime"`      //交易时间
	IdKey       string `json:"idKey"`       //对象存储的key
	FileId      string `json:"fileId"`      //文件Id
	FileName    string `json:"fileName"`    //文件名
	FileSrcName string `json:"fileSrcName"` //源文件名
	FilePath    string `json:"filePath"`    //文件存储路径
	FileURL     string `json:"fileURL"`     //文件访问URL
	FileType    string `json:"fileType"`    //文件类型,扩展名
	FileSize    int    `json:"fileSize"`    //文件大小
	FileHash    string `json:"fileHash"`    //文件Hash
}

type PublicKey struct {
	ModelName string
	PublicKey string
	MainId    string
}

type Cryto struct {
	MainId    string `json:"mainId"`
	ModelName string `json:"modelName"`
}

type PUB_KEY struct {
	PUB_KEY string `json:"PUB_KEY"` //PUB_KEY
}

type pList []RBCTable

func (p pList) Len() int {
	return len(p)
}

func (p pList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p pList) Less(i, j int) bool {
	tab1 := p[i]
	tab2 := p[j]

	return tab1.TableName < tab2.TableName
}

func CleanCache(stub shim.ChaincodeStubInterface) {
	chanModelDatas.Lock()
	defer chanModelDatas.Unlock()

	chr, err := stub.GetChannelHeader()
	if err != nil {
		log.Logger.Errorf("CleanCache GetChannelHeader err : %s", err)
	}

	if chanModelDatas.mapDataBuf[chr.ChannelId] != nil {
		delete(chanModelDatas.mapDataBuf, chr.ChannelId)
	}
}

func LoadCache(stub shim.ChaincodeStubInterface) {

	chanModelDatas.Lock()
	defer chanModelDatas.Unlock()

	chr, err := stub.GetChannelHeader()
	if err != nil {
		log.Logger.Errorf("LoadCache GetChannelHeader err : %s", err)
	}

	modelData := ModelDataStruct{modelsCache: make([]*RBCModelInfo, 0), tablesCache: map[string][]*RBCTable{}}
	chanModelDatas.mapDataBuf[chr.ChannelId] = &modelData

	size := stub.Size("__RBC_MODEL_")

	var models []*RBCModelInfo
	tables := map[string][]*RBCTable{}

	for i := 0; i < size; i++ {
		modelkey := stub.Get("__RBC_MODEL_", i)

		modelBytes, err := stub.GetState(modelkey)
		if err != nil {
			log.Logger.Error("stub getstate modelkey : %s err : %s", modelkey, err)
			break
		}

		model := &RBCModelInfo{}
		err = jsoniter.Unmarshal(modelBytes, model)
		if err != nil {
			log.Logger.Errorf("unmarshal model err : %s", err)
		}
		models = append(models, model)
	}

	if len(models) > 0 {
		for i := 0; i < len(models); i++ {
			model := models[i]
			rlistName := "__RBC_MODEL_" + model.ModelName + "_"

			var modelTables []*RBCTable
			size := stub.Size(rlistName)
			for j := 0; j < size; j++ {
				table := &RBCTable{}
				tablekey := stub.Get(rlistName, j)
				tableBytes, err := stub.GetState(tablekey)
				if err != nil {
					log.Logger.Error("stub getstate tablekey : %s err : %s", tablekey, err)
					break
				}

				err = jsoniter.Unmarshal(tableBytes, table)
				if err != nil {
					log.Logger.Errorf("unmarshal model err : %s", err)
				}
				modelTables = append(modelTables, table)
			}
			tables[model.ModelName] = modelTables
		}
	}

	if len(models) > 0 {
		modelData.modelsCache = models
	}

	if len(tables) > 0 {
		modelData.tablesCache = tables
	}

}

// Init initializes the sample system chaincode by storing the key and value
// arguments passed in as parameters
func (t *RBCModel) Init(_ shim.ChaincodeStubInterface) pb.Response {
	//as system chaincodes do not take part in consensus and are part of the system,
	//best practice to do nothing (or very little) in Init.

	defaultExpiration, _ := time.ParseDuration("1800s")
	gcInterval, _ := time.ParseDuration("60s")
	t.cryptoCache = cache.NewCache(defaultExpiration, gcInterval)

	return shim.Success(nil)
}

// Invoke gets the supplied key and if it exists, updates the key with the newly
// supplied value.
func (t *RBCModel) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	chdr, err := stub.GetChannelHeader()
	if err != nil {
		return shim.Error(err.Error())
	}
	if chanModelDatas.mapDataBuf[chdr.ChannelId] == nil {
		LoadCache(stub)
	}

	args := stub.GetArgs()
	if len(args) < 2 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
	}

	f := string(args[1])
	switch f {
	case QueryRBCCName:
		return shim.Success([]byte("rbcmodel"))
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
	case SetModel:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.setModel(stub, args[2])
	case GetModel:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}
		return t.getModel(stub, string(args[2]))
	case DeleteModel:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.deleteModel(stub, string(args[2]))
	case GetModelList:
		return t.getModelList(stub)
	case SetTable:
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		err := t.checkModelExist(stub, string(args[2]))
		if err != nil {
			return shim.Error(fmt.Sprintf("model is not exist err: %s", err))
		}
		return t.setTable(stub, string(args[2]), string(args[3]))
	case DeleteTable:
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}
		customerEntity, err := t.getCustomerEntity(stub)
		if err != nil {
			return shim.Error(err.Error())
		}
		if customerEntity == nil || customerEntity.CustomerType != "1" { //非超级会员不能修改模型
			return shim.Error("deleteTable must from super customer")
		}

		err = t.checkModelExist(stub, string(args[2]))
		if err != nil {
			return shim.Error(fmt.Sprintf("model is not exist err: %s", err))
		}

		return t.deleteTable(stub, string(args[2]), string(args[3]))
	case GetTable:
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		err := t.checkModelExist(stub, string(args[2]))
		if err != nil {
			return shim.Error(fmt.Sprintf("model is not exist err: %s", err))
		}

		return t.getTable(stub, string(args[2]), string(args[3]))
	case GetTableList: //table list 获取 所有表目录、表索引
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		err := t.checkModelExist(stub, string(args[2]))
		if err != nil {
			return shim.Error(fmt.Sprintf("model is not exist err: %s", err))
		}

		return t.getTableList(stub, string(args[2]))
	case GetTableListWithPageCat: //table list 获取 所有表目录、表索引
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.getTableListWithPageCat(stub, string(args[2]))
	case SetMainPublicKey:

		if len(args) == 3 {
			return t.setMainPublicKeyNew(stub, string(args[2]))
		} else if len(args) == 5 {
			return t.setMainPublicKey(stub, string(args[3]), args[4])
		} else {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

	case GetMainPublicKey:
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		return t.getMainPublicKeyNew(stub, string(args[2]), string(args[3]))

	case GetMainCryptogram:
		if len(args) < 6 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}

		//对合约控制表的权限特殊处理,处理合约与模型的对应关系
		if "__CHAINCODE" == string(args[3]) {
			chainCodeKey := "__RBC_MODEL_CHAINCODE_" + string(args[2])
			modelBuf, err := stub.GetState(chainCodeKey)
			if err != nil {
				return shim.Success([]byte("NOMODEL"))
			}

			args[2] = modelBuf
		}

		err := t.checkModelExist(stub, string(args[2]))
		if err != nil {
			return shim.Success([]byte("NOMODEL"))
		}
		return t.getMainCryptogram(stub, string(args[2]), string(args[3]), string(args[4]), string(args[5]))
	case GetFromHttp:
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}
		return t.getFromHttp(stub, string(args[2]), string(args[3]))
	case GetAttach:
		if len(args) < 3 {
			return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
		}
		return t.getAttach(stub, string(args[2]))

	default:
		jsonResp := fmt.Sprintf("function %s is not found", f)
		return shim.Error(jsonResp)
	}
}

func (t *RBCModel) setModel(stub shim.ChaincodeStubInterface, modelInfo []byte) pb.Response {

	rbcModel := &RBCModelInfo{}
	err := jsoniter.Unmarshal(modelInfo, rbcModel)
	if err != nil {
		return shim.Error(fmt.Sprintf("args[2] parse to rbcmodel has err %s", err))
	}

	if len(rbcModel.ModelName) < 1 {
		return shim.Error("rbcmodel name is empty")
	}

	customerEntity, err  := t.getCustomerEntity(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if customerEntity == nil {
		return shim.Error(fmt.Sprintf("customerEntity is nil"))
	}

	key := "__RBC_MODEL_" + rbcModel.ModelName

	rbcModel.IdKey = key
	rbcModel.TxId = stub.GetTxID()
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	rbcModel.TxTime = txTimeStr

	rbcModelBytes, err := jsoniter.Marshal(rbcModel)
	if err != nil {
		return shim.Error(fmt.Sprintf("rbc model marshal to bytes err: %s", err))
	}

	stub.PutState(key, rbcModelBytes)
	stub.Add("__RBC_MODEL_", key)
	CleanCache(stub)

	return shim.Success([]byte("setModel success"))
}

func (t *RBCModel) deleteModel(stub shim.ChaincodeStubInterface, modelName string) pb.Response {

	customerEntity, err := t.getCustomerEntity(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if customerEntity == nil || customerEntity.CustomerType != "1" { //非超级会员不能修改模型
		return shim.Error("deleteModel must from super customer")
	}

	err = t.checkModelExist(stub, modelName)
	if err != nil {
		return shim.Error(fmt.Sprintf("model is not exist err: %s", err))
	}

	key := "__RBC_MODEL_" + modelName
	tableAmount := stub.Size(key + "_")
	if tableAmount > 0 {
		return shim.Error(fmt.Sprintf("model %s exist table can't delete", modelName))
	}
	stub.DelState(key)
	stub.Remove("__RBC_MODEL_", key)
	CleanCache(stub)

	return shim.Success([]byte("delete model success"))
}

// 获取所有表目录
func (t *RBCModel) getModel(stub shim.ChaincodeStubInterface, modelName string) pb.Response {

	err := t.checkModelExist(stub, modelName)
	if err != nil {
		return shim.Error(fmt.Sprintf("model is not exist err: %s", err))
	}

	chanModelDatas.Lock()
	defer chanModelDatas.Unlock()

	chr, _ := stub.GetChannelHeader()
	if chanModelDatas.mapDataBuf[chr.ChannelId] != nil {
		modelData := chanModelDatas.mapDataBuf[chr.ChannelId]
		if modelData != nil {
			modelsCache := modelData.modelsCache
			if modelsCache != nil {
				for _, model := range modelsCache {
					if model.ModelName == modelName {
						model.TableAmount = len(modelData.tablesCache[modelName])
						modelBuf, err := jsoniter.Marshal(model)
						if err != nil {
							return shim.Error(fmt.Sprintf("existModel marshal err: %s", err))
						}

						return shim.Success(modelBuf)
					}
				}
			}
		}
	}

	key := "__RBC_MODEL_" + modelName

	modelBuf, err := stub.GetState(key)

	if err != nil {
		return shim.Error(fmt.Sprintf("rbc model get err: %s", err))
	}
	existModel := &RBCModelInfo{}
	err = jsoniter.Unmarshal(modelBuf, existModel)

	if err != nil {
		return shim.Error(fmt.Sprintf("rbc model get err: %s", err))
	}
	tableAmount := stub.Size(key + "_")
	if tableAmount < 0 {
		tableAmount = 0
	}
	existModel.TableAmount = tableAmount
	modelBuf, err = jsoniter.Marshal(existModel)
	if err != nil {
		return shim.Error(fmt.Sprintf("existModel marshal err: %s", err))
	}

	return shim.Success(modelBuf)
}

func (t *RBCModel) checkModelExist(stub shim.ChaincodeStubInterface, modelName string) error {

	chanModelDatas.Lock()
	defer chanModelDatas.Unlock()

	if len(modelName) < 1 {
		return fmt.Errorf("model name is empty")
	}

	chr, _ := stub.GetChannelHeader()
	if chanModelDatas.mapDataBuf[chr.ChannelId] != nil {
		modelData := chanModelDatas.mapDataBuf[chr.ChannelId]
		if modelData != nil {
			modelsCache := modelData.modelsCache
			if modelsCache != nil && len(modelsCache) > 0 {
				for _, model := range modelsCache {
					if model.ModelName == modelName {
						return nil
					}
				}
			}
		}
	}

	modelBytes, _ := stub.GetState("__RBC_MODEL_" + modelName)
	if modelBytes == nil || len(modelBytes) < 1 {
		return fmt.Errorf("model %s is not exist", modelName)
	}

	return nil
}

// 新建表
func (t *RBCModel) setTable(stub shim.ChaincodeStubInterface, modelName, params string) pb.Response {
	var rbcTableEntity RBCTable
	err := jsoniter.Unmarshal([]byte(params), &rbcTableEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("createTableInfo param is not a json, %s", err))
	}
	//colist字段不为空，解析含有Column字段json
	if rbcTableEntity.ColList != nil {
		var f RBCColumn
		err := jsoniter.Unmarshal([]byte(params), &f)
		if err != nil {
			return shim.Error(fmt.Sprintf("createTableInfo unmarshal failure, %s", err))
		}
	}

	key := "__RBC_MODEL_" + modelName + "_" + rbcTableEntity.TableName

	rbcTableEntity.IdKey = key
	rbcTableEntity.TxId = stub.GetTxID()
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	rbcTableEntity.TxTime = txTimeStr

	modelBytes, err := jsoniter.Marshal(rbcTableEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("rbcTableEntity Marshal 错误"))
	}

	//对合约控制表的权限特殊处理,处理合约与模型的对应关系
	if rbcTableEntity.TableName == "__CHAINCODE" {
		for _, colEntity := range rbcTableEntity.ColList {
			chainCodeKey := "__RBC_MODEL_CHAINCODE_" + colEntity.ColumnName
			stub.PutState(chainCodeKey, []byte(modelName))
		}
	}

	//维护索引

	stub.PutState(key, modelBytes)
	//维护列表
	stub.Add("__RBC_MODEL_"+modelName+"_", key)

	//表结构发生变化时清空缓存
	t.cryptoCache.Flush()
	CleanCache(stub)
	return shim.Success(modelBytes)
}

// 删除表
func (t *RBCModel) deleteTable(stub shim.ChaincodeStubInterface, modelName, tableName string) pb.Response {
	key := "__RBC_MODEL_" + modelName + "_" + tableName

	stub.DelState(key)
	stub.Remove("__RBC_MODEL_"+modelName+"_", key)

	//表结构发生变化时清空缓存
	t.cryptoCache.Flush()
	CleanCache(stub)
	return shim.Success([]byte("delete table success"))
}

// 获取所有表目录
func (t *RBCModel) getModelList(stub shim.ChaincodeStubInterface) pb.Response {

	chanModelDatas.Lock()
	defer chanModelDatas.Unlock()

	rListName := "__RBC_MODEL_"

	nSize := stub.Size(rListName)
	cpno := 1

	listValue := make([]interface{}, 0)
	pageList := &shim.PageList{Cpno: strconv.Itoa(cpno), Rnum: strconv.Itoa(nSize), List: listValue}

	chr, _ := stub.GetChannelHeader()
	if chanModelDatas.mapDataBuf[chr.ChannelId] != nil {
		modelData := chanModelDatas.mapDataBuf[chr.ChannelId]
		if modelData != nil {
			modelsCache := modelData.modelsCache
			if modelsCache != nil && len(modelsCache) > 0 {
				for i := 0; i < len(modelsCache); i++ {
					model := modelsCache[i]
					model.TableAmount = len(modelData.tablesCache[model.ModelName])
					pageList.List = append(pageList.List, model)
				}
			}
		}
	}

	if len(pageList.List) == 0 {
		//循环获取每页表信息
		for i := 0; i < nSize; i++ {
			existModel := &RBCModelInfo{}
			modelKey := stub.Get(rListName, i)
			if len(modelKey) > 0 {
				existBytes, _ := stub.GetState(modelKey)
				if existBytes != nil {
					err := jsoniter.Unmarshal(existBytes, existModel)
					if err != nil {
						continue
					}

					tableAmount := stub.Size(modelKey + "_")
					if tableAmount < 0 {
						tableAmount = 0
					}
					existModel.TableAmount = tableAmount
					pageList.List = append(pageList.List, existModel)

				}
			}
		}
	}
	pageListBuf, _ := jsoniter.Marshal(pageList)
	return shim.Success(pageListBuf)
}

// 获取所有表目录
func (t *RBCModel) getTableList(stub shim.ChaincodeStubInterface, modelName string) pb.Response {

	chanModelDatas.Lock()
	defer chanModelDatas.Unlock()

	//bTime := time.Now().UnixNano()
	rListName := "__RBC_MODEL_" + modelName + "_"

	nSize := stub.Size(rListName)
	cpno := 1
	listValue := make([]interface{}, 0)
	pageList := &shim.PageList{Cpno: strconv.Itoa(cpno), Rnum: strconv.Itoa(nSize), List: listValue}

	tables := make([]*RBCTable, 0)

	chr, _ := stub.GetChannelHeader()
	if chanModelDatas.mapDataBuf[chr.ChannelId] != nil {
		modelData := chanModelDatas.mapDataBuf[chr.ChannelId]
		if modelData != nil {
			tablesCache := modelData.tablesCache
			if len(tablesCache) > 0 {
				tables = tablesCache[modelName]
			}
		}
	}

	//jsTime := time.Now().UnixNano()

	//var tableTime, regTime int64
	if len(tables) > 0 {
		//log.Logger.Infof("model name : %s size : %d", modelName, nSize)
		for i := 0; i < len(tables); i++ {
			table := tables[i]
			tableIn := *table
			tableIn.ColList = nil
			tableIn.IndexList = nil
			pageList.List = append(pageList.List, tableIn)
		}
		//tableTime = time.Now().UnixNano()
	} else {
		//循环获取每页表信息
		for i := 0; i < nSize; i++ {
			existTable := &RBCTable{}
			tableNum := stub.Get(rListName, i)
			if len(tableNum) > 0 {
				existBytes, _ := stub.GetState(tableNum)
				if existBytes != nil {
					err := jsoniter.Unmarshal(existBytes, existTable)
					if err != nil {
						return shim.Error(fmt.Sprintf("json Unmarshal error : %s", err))
					}

					existTable.ColList = nil
					existTable.IndexList = nil

					pageList.List = append(pageList.List, existTable)
				}
			}
		}

		//regTime = time.Now().UnixNano()
	}

	pageListBuf, _ := jsoniter.Marshal(pageList)
	//log.Logger.Infof("pagelistbuf : %s", string(pageListBuf))
	//eTime := time.Now().UnixNano()
	//
	//log.Logger.Infof("len pageListBuf : %d json marsh duration : %d, from cache duration : %d, from db duration : %d,  hole duration : %d",
	//	len(pageListBuf), jsTime-bTime, tableTime-jsTime, regTime-jsTime, eTime-bTime)
	//
	//defer func() {
	//	log.Logger.Infof("defer log len pageListBuf : %d json marsh duration : %d, from cache duration : %d, from db duration : %d,  hole duration : %d",
	//		len(pageListBuf), jsTime-bTime, tableTime-jsTime, regTime-jsTime, eTime-bTime)
	//}()

	return shim.Success(pageListBuf)
}

func (t *RBCModel) getTableListWithPageCat(stub shim.ChaincodeStubInterface, params string) pb.Response {

	var m map[string]string
	err := jsoniter.Unmarshal([]byte(params), &m)

	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	modelName := m["modelName"]
	pageNo := m["cpno"]
	category := m["category"]
	var havecol = false
	if m["havecol"] != "" {
		havecol, _ = strconv.ParseBool(m["havecol"])
	}

	err = t.checkModelExist(stub, modelName)
	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	rListName := "__RBC_MODEL_" + modelName + "_"

	nSize := stub.Size(rListName)
	if pageNo == "" {
		pageNo = "0"
	}
	cpno, err := strconv.Atoi(pageNo)

	if err != nil {
		shim.Error("fail to recast pageNo!")
	}

	listValue := make([]interface{}, 0)
	pageList := &shim.PageList{Cpno: strconv.Itoa(cpno), Rnum: strconv.Itoa(nSize), List: listValue}

	pls := make(pList, 0)
	tables := make([]*RBCTable, 0)

	chanModelDatas.Lock()
	defer chanModelDatas.Unlock()

	chr, _ := stub.GetChannelHeader()
	if chanModelDatas.mapDataBuf[chr.ChannelId] != nil {
		modelData := chanModelDatas.mapDataBuf[chr.ChannelId]
		if modelData != nil {
			tablesCache := modelData.tablesCache
			if len(tablesCache) > 0 {
				tables = tablesCache[modelName]
			}
		}
	}

	if len(tables) > 0 {
		for i := 0; i < len(tables); i++ {
			table := tables[i]
			tableIn := *table

			if havecol == false {
				tableIn.ColList = nil
				tableIn.IndexList = nil
			}

			if category != "" {
				categories := strings.Split(category, "~")
				if len(categories) == 1 {
					if categories[0] == table.TableCategory {
						pls = append(pls, tableIn)
					}
				} else if len(categories) == 2 {
					if categories[0] == table.TableCategory &&
						categories[1] == table.TableCategory1 {
						pls = append(pls, tableIn)
					}
				} else if len(categories) == 3 {
					if categories[0] == table.TableCategory &&
						categories[1] == table.TableCategory1 &&
						categories[2] == table.TableCategory2 {
						pls = append(pls, tableIn)
					}
				}
			} else {
				pls = append(pls, tableIn)
			}
		}
	} else {
		for i := 0; i < nSize; i++ {
			existTable := &RBCTable{}
			tableNum := stub.Get(rListName, i)
			if len(tableNum) > 0 {
				existBytes, _ := stub.GetState(tableNum)
				if existBytes != nil {
					err := jsoniter.Unmarshal(existBytes, existTable)

					if err == nil {
						if havecol == false {
							existTable.ColList = nil
							existTable.IndexList = nil
						}

						if category != "" {
							categories := strings.Split(category, "~")
							if len(categories) == 1 {
								if categories[0] == existTable.TableCategory {
									pls = append(pls, *existTable)
								}
							} else if len(categories) == 2 {
								if categories[0] == existTable.TableCategory &&
									categories[1] == existTable.TableCategory1 {
									pls = append(pls, *existTable)
								}
							} else if len(categories) == 3 {
								if categories[0] == existTable.TableCategory &&
									categories[1] == existTable.TableCategory1 &&
									categories[2] == existTable.TableCategory2 {
									pls = append(pls, *existTable)
								}
							}
						} else {
							pls = append(pls, *existTable)
						}
					}
				}
			}
		}
	}

	sort.Sort(pls)
	catListSize := len(pls)
	if cpno < 0 {
		cpno = 0
	}

	if cpno > catListSize/100 {
		cpno = catListSize / 100
	}

	bNo := cpno * 100
	eNo := (cpno + 1) * 100

	if eNo > catListSize {
		eNo = catListSize
	}

	//循环获取每页表信息
	for i := bNo; i < eNo; i++ {
		pageList.List = append(pageList.List, pls[i])
	}

	pageList.Rnum = strconv.Itoa(catListSize)
	pageList.Cpno = strconv.Itoa(cpno)
	pageListBuf, _ := jsoniter.Marshal(pageList)

	//plsBuf, _ := jsoniter.Marshal(pls)
	//log.Logger.Infof("pagelistBuf : %s pls : %s", string(pageListBuf), string(plsBuf))

	return shim.Success(pageListBuf)
}

// 查询单张表信息
func (t *RBCModel) getTable(stub shim.ChaincodeStubInterface, modelName, tableName string) pb.Response {

	chanModelDatas.Lock()
	defer chanModelDatas.Unlock()

	var tableBuf []byte
	var err error

	chr, _ := stub.GetChannelHeader()
	if chanModelDatas.mapDataBuf[chr.ChannelId] != nil {
		modelData := chanModelDatas.mapDataBuf[chr.ChannelId]
		if modelData != nil {
			tablesCache := modelData.tablesCache[modelName]
			if len(tablesCache) > 0 {
				for i := 0; i < len(tablesCache); i++ {
					table := tablesCache[i]
					if table.TableName == tableName {
						tableBuf, err = jsoniter.Marshal(table)
						if err != nil {
							return shim.Error(fmt.Sprintf(" jsoniter.Marshal(table) err : %s", err))
						}

						return shim.Success(tableBuf)
					}
				}
			}
		}
	}

	key := "__RBC_MODEL_" + modelName + "_" + tableName
	tableBuf, err = stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("get table with key : %s err : %s", key, err))
	}

	return shim.Success(tableBuf)
}

func (t *RBCModel) queryStateRList(stub shim.ChaincodeStubInterface, rListName string, params string) pb.Response {
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

//设置主体公钥
func (t *RBCModel) setMainPublicKey(stub shim.ChaincodeStubInterface, mainId string, publicKey []byte) pb.Response {
	if len(mainId) < 1 || publicKey == nil || len(publicKey) < 65 {
		return shim.Error("mainId is empty")
	}

	key := "__RBC_MODEL_PUB_" + mainId
	stub.PutState(key, publicKey)
	return shim.Success([]byte("set Main PublicKey success"))
}

////设置主体公钥
func (t *RBCModel) setMainPublicKeyNew(stub shim.ChaincodeStubInterface, params string) pb.Response {

	var keys []PublicKey
	err := jsoniter.Unmarshal([]byte(params), &keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("Unmarshal param  err: %s", err))
	}

	for _, key := range keys {
		keyIn := "__RBC_MODEL_PUB_" + key.MainId
		err = stub.PutState(keyIn, []byte(key.PublicKey))
		if err != nil {
			return shim.Error(fmt.Sprintf("stub putstate err: %s", err))
		}
	}

	return shim.Success([]byte("set Main PublicKey success"))
}

//获取主体公钥
func (t *RBCModel) getMainPublicKey(stub shim.ChaincodeStubInterface, modelName, mainId string) pb.Response {

	if len(mainId) < 1 {
		return shim.Error("mainId is empty")
	}

	key := "__RBC_MODEL_PUB_" + mainId

	bReturn, err := stub.GetState(key)
	if err != nil || bReturn == nil || len(bReturn) < 65 {
		mRes := t.getModel(stub, modelName)
		existModel := &RBCModelInfo{}
		err = jsoniter.Unmarshal(mRes.Payload, existModel)

		if err != nil {
			return shim.Error(fmt.Sprintf("rbc model get err: %s", err))
		}

		//从解密中心重新注册公钥
		chr, _ := stub.GetChannelHeader()
		cid := chr.ChannelId
		rbcMessage := &common.RBCMessage{ChainID: cid, Type: 21}
		postURL := strings.TrimSpace(existModel.DecryptService)
		//postURL := "http://192.168.1.34:8080/rbc.crypto/rbc/crypto/decryptService.htm"

		if len(postURL) < 5 {
			return shim.Error("rbc model DecryptService is empty")
		}

		httpRequest := &common.RBCHttpRequest{Type: "", Method: "POST", Endpoint: postURL}
		signProp, err := stub.GetSignedProposal()
		if err != nil {
			return shim.Error(fmt.Sprintf("get sign prop has err:%s", err))
		}
		signPropBuf, err := signProp.Marshal()
		if err != nil {
			return shim.Error(fmt.Sprintf("get sign prop has err:%s", err))
		}

		params := make(map[string]string)
		params["method"] = "getMainPublicKey"
		params["signPropBuf"] = base64.StdEncoding.EncodeToString(signPropBuf)
		params["modelName"] = modelName
		params["mainId"] = mainId
		params["NOSESSION_PASS"] = "1"

		httpRequest.Params = params
		reqBuf, err := httpRequest.Marshal()
		if err != nil {
			return shim.Error(fmt.Sprintf("httpRequest marphal err: %s", err))
		}

		rbcMessage.Data = reqBuf
		res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
		if err != nil {
			return shim.Error(fmt.Sprintf("create Main Publick ID Http err : %s", err))
		}

		var jPubKey = struct {
			PUB_KEY  string `json:"PUB_KEY"`  //PUB_KEY
			NEEDSAVE string `json:"NEEDSAVE"` //NEEDSAVE
		}{}

		err = jsoniter.Unmarshal(res.Data, &jPubKey)
		if err != nil {
			log.Logger.Errorf("httpRequest string is %s rbcMessage string is %s get Value is %s",
				httpRequest.String(), rbcMessage.String(), string(res.Data))
			return shim.Error(fmt.Sprintf("create Main Publick ID JSON err : %s", err))
		}

		if len(jPubKey.PUB_KEY) < 1 {
			return shim.Error("create Main Publick ID return empty ")
		}
		return shim.Success([]byte(jPubKey.PUB_KEY))
	}

	return shim.Success(bReturn)

}

//获取主体公钥
func (t *RBCModel) getMainPublicKeyNew(stub shim.ChaincodeStubInterface, modelName, reqParam string) pb.Response {

	bTime := time.Now().UnixNano() / 1000000
	isVal := strings.HasPrefix(reqParam, "{")

	var mainId string
	var getKeys map[string]Cryto
	var bReturn []byte

	if isVal {

		err := jsoniter.Unmarshal([]byte(reqParam), &getKeys)
		if err != nil {
			return shim.Error(fmt.Sprintf("getKeys Unmarshal err: %s", err))
		}

		pubKeys := map[string]*PUB_KEY{}
		paramsMap := map[string]interface{}{}

		for k, v := range getKeys {
			key := "__RBC_MODEL_PUB_" + v.MainId

			bReturn, err := stub.GetState(key)
			if err != nil || bReturn == nil || len(bReturn) < 65 {
				paramsMap[k] = v
			} else {
				pubKey := &PUB_KEY{PUB_KEY: string(bReturn)}
				pubKeys[k] = pubKey
			}
		}
		pTime1 := time.Now().UnixNano() / 1000000
		if len(paramsMap) > 0 {
			//从解密中心重新注册公钥
			chr, _ := stub.GetChannelHeader()
			cid := chr.ChannelId

			mRes := t.getModel(stub, modelName)
			existModel := &RBCModelInfo{}
			err = jsoniter.Unmarshal(mRes.Payload, existModel)
			if err != nil {
				return shim.Error(fmt.Sprintf("rbc model get err: %s", err))
			}

			postURL := strings.TrimSpace(existModel.DecryptService)

			httpRequest := &common.RBCHttpRequest{Type: "", Method: "POST", Endpoint: postURL}
			bytePl, _ := jsoniter.Marshal(paramsMap)

			params := make(map[string]string)
			params["method"] = "getMainPublicKey"
			params["reqInfo"] = string(bytePl)
			params["NOSESSION_PASS"] = "1"
			//params["signPropBuf"] = base64.StdEncoding.EncodeToString(signPropBuf)
			httpRequest.Params = params

			reqBuf, err := httpRequest.Marshal()
			if err != nil {
				return shim.Error(fmt.Sprintf("httpRequest marphal err: %s", err))
			}

			rbcMessage := &common.RBCMessage{ChainID: cid, Type: 21}
			rbcMessage.Data = reqBuf
			res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
			if err != nil || res == nil {
				return shim.Error(fmt.Sprintf("getMainPublicKeyNew create Main Publick ID Http err : %s", err))
			}

			respMap := map[string]*PUB_KEY{}

			log.Logger.Warnf("rbcmessage is : %s resp map data : %s", rbcMessage.String(), string(res.Data))

			err = jsoniter.Unmarshal(res.Data, &respMap)
			if err != nil {
				log.Logger.Errorf("get Value is err %s", string(res.Data))
				return shim.Error(fmt.Sprintf("getMainPublicKeyNew create Main Publick ID JSON err : %s res Data: %s", err, string(res.Data)))
			}

			for k, v := range respMap {
				pubKeys[k] = v
			}
		}

		eTime := time.Now().UnixNano() / 1000000

		timeStr := fmt.Sprintf("getMainPubKeyNew Use Time %d ,http %d use %d , get state use time %d",
			eTime-bTime, len(paramsMap), eTime-pTime1, pTime1-bTime)

		pubKeys["time"] = &PUB_KEY{timeStr}

		bReturn, _ = jsoniter.Marshal(pubKeys)

		if eTime-bTime > 500 {
			log.Logger.Warnf("getMainPubKeyNew Use Time %d ,http %d use %d , get state use time %d",
				eTime-bTime, len(paramsMap), eTime-pTime1, pTime1-bTime)
		}

		return shim.Success(bReturn)

	} else {
		mainId = reqParam

		return t.getMainPublicKey(stub, modelName, mainId)
	}
}

//获取主体解密密码
func (t *RBCModel) getMainCryptogram(stub shim.ChaincodeStubInterface, modelName, tableName, pubG, reqInfo string) pb.Response {
	if len(reqInfo) < 1 {
		return shim.Error("mainName tableName mainId pubR oneof is empty")
	}
	if len(pubG) < 1 {
		pubG = ""
	}
	bTime := time.Now().UnixNano() / 1000000

	mRes := t.getModel(stub, modelName)
	existModel := &RBCModelInfo{}
	err := jsoniter.Unmarshal(mRes.Payload, existModel)

	if err != nil {
		return shim.Error(fmt.Sprintf("rbc model get err: %s", err))
	}

	createBytes, _ := stub.GetCreator()
	chr, _ := stub.GetChannelHeader()

	chaincodeSpec := &pb.ChaincodeSpec{}
	if err := chaincodeSpec.Unmarshal(chr.Extension); err != nil {
		return shim.Error("get chaincode spec has err")
	}

	chaincodeBuf, _ := chaincodeSpec.ChaincodeId.Marshal()

	//同一个合约、同一个人获取的密码是一套缓存
	md5Ctx := md5.New()
	md5Ctx.Write(chaincodeBuf)
	md5Ctx.Write(createBytes)
	md5Ctx.Write([]byte(modelName))
	md5Ctx.Write([]byte(tableName))
	md5Ctx.Write([]byte(pubG))
	md5Ctx.Write([]byte(reqInfo))
	cacheKey := hex.EncodeToString(md5Ctx.Sum(nil))

	if exist, found := t.cryptoCache.Get(cacheKey); found {
		cacheRes := exist.(*orderer.BroadcastResponse)
		//log.Logger.Infof("use cache gram %s", cacheKey)
		return shim.Success(cacheRes.Data)
	}

	envSign := ""

	if t.signFunc != nil {
		dockerHubDomain := viper.GetString("docker.hub.domain")
		if len(dockerHubDomain) < 10 {
			dockerHubDomain = ""
		} else {
			dockerHubDomain = strings.TrimSpace(dockerHubDomain)
		}
		dockerImageTag := viper.GetString("docker.image.tag")
		javaEnv := dockerHubDomain + "rongzer/blockchain-javaenv" + dockerImageTag

		envSign = t.signFunc.(func(str, javaEnv string) string)(stub.GetTxID(), javaEnv)
		//log.Logger.Infof("get gram env %s", envSign)
	} else {
		//log.Logger.Infof("get gram env func is null")
	}

	//从解密中心重新注册公钥
	cid := chr.ChannelId

	rbcMessage := &common.RBCMessage{ChainID: cid, Type: 21}
	postURL := strings.TrimSpace(existModel.DecryptService)
	if len(postURL) < 5 {
		return shim.Error("rbc model DecryptService is empty")
	}

	httpRequest := &common.RBCHttpRequest{Type: "", Method: "POST", Endpoint: postURL}

	signProp, err := stub.GetSignedProposal()
	if err != nil {
		return shim.Error(fmt.Sprintf("get sign prop has err:%s", err))
	}
	signPropBuf, err := signProp.Marshal()
	if err != nil {
		return shim.Error(fmt.Sprintf("get sign prop has err:%s", err))
	}

	params := make(map[string]string)
	params["method"] = "getMainCryptogram"

	params["signPropBuf"] = base64.StdEncoding.EncodeToString(signPropBuf)
	params["channelId"] = chr.ChannelId
	params["modelName"] = modelName
	params["tableName"] = tableName
	params["pubG"] = pubG
	params["reqInfo"] = reqInfo
	params["envSign"] = envSign
	params["NOSESSION_PASS"] = "1"

	httpRequest.Params = params
	reqBuf, err := httpRequest.Marshal()
	if err != nil {
		return shim.Error(fmt.Sprintf("httpRequest marphal err: %s", err))
	}
	rbcMessage.Data = reqBuf
	pTime1 := time.Now().UnixNano() / 1000000

	res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
	eTime := time.Now().UnixNano() / 1000000
	if eTime-bTime > 500 {

		log.Logger.Warnf("getMainCryptogram Use Time %d ,http Time %d", eTime-bTime, eTime-pTime1)
	}

	if err != nil {
		return shim.Error(fmt.Sprintf("getMainCryptogram err : %s", err))
	}

	if res == nil || res.Data == nil {
		return shim.Error(fmt.Sprintf("getMainCryptogram is nil"))
	}

	return shim.Success(res.Data)
}

func (t *RBCModel) getCustomerEntity(stub shim.ChaincodeStubInterface) (*rbccustomer.CustomerEntity, error) {
	//取当前用户
	serializedIdentity := &msp.SerializedIdentity{}
	createBytes, err := stub.GetCreator()
	if err != nil {
		return nil, err
	}

	if err := serializedIdentity.Unmarshal(createBytes); err != nil {
		return nil, err
	}

	md5Ctx := md5.New()

	md5Ctx.Write([]byte(strings.TrimSpace(string(serializedIdentity.IdBytes))))
	md5Str := hex.EncodeToString(md5Ctx.Sum(nil))

	//从install中获取chaincode的内容
	invokeArgs := []string{"query", "queryOne", md5Str}

	chr, _ := stub.GetChannelHeader()
	resp := stub.InvokeChaincode("rbccustomer", util.ArrayToChaincodeArgs(invokeArgs), chr.ChannelId)
	if resp.Status != shim.OK {
		log.Logger.Errorf("get customerEntity is err %s", resp.Message)
		return nil, errors.New(fmt.Sprint("get customerEntity err ", resp.Message))
	}

	//将chaincode的代码设置到参数中(ChainCodeData)
	customerEntity := &rbccustomer.CustomerEntity{}
	if err := jsoniter.Unmarshal(resp.Payload, customerEntity); err != nil {
		log.Logger.Errorf("resp.Payload unmarshal err : %s", err)
		log.Logger.Infof("IdBytes : %s md5Str : %s resp str : %s", string(serializedIdentity.IdBytes), md5Str, string(resp.Payload))
		return nil, err
	}

	return customerEntity, nil
}

//通过orderer发送通用http请求
func (t *RBCModel) getFromHttp(stub shim.ChaincodeStubInterface, url, param string) pb.Response {
	chr, _ := stub.GetChannelHeader()

	rbcMessage := &common.RBCMessage{ChainID: chr.ChannelId, Type: 21}
	postURL := strings.TrimSpace(url)
	if len(postURL) < 5 {
		return shim.Error("post url is empty")
	}

	httpRequest := &common.RBCHttpRequest{Type: "", Method: "POST", Endpoint: postURL}
	if len(param) > 0 {
		var params map[string]string
		err := jsoniter.Unmarshal([]byte(param), &params)
		if err != nil {
			return shim.Error(fmt.Sprintf("params Unmarshal err: %s", err))
		}
		httpRequest.Params = params
	}

	reqBuf, err := httpRequest.Marshal()
	if err != nil {
		return shim.Error(fmt.Sprintf("httpRequest marphal err: %s", err))
	}
	rbcMessage.Data = reqBuf

	res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
	if err != nil {
		return shim.Error(fmt.Sprintf("getFromHttp from orderer err: %s", err))
	}
	return shim.Success(res.Data)
}

//获取clob字段的详细信息
func (t *RBCModel) getAttach(stub shim.ChaincodeStubInterface, key string) pb.Response {
	chr, _ := stub.GetChannelHeader()

	rbcMessage := &common.RBCMessage{ChainID: chr.ChannelId, Type: 24}
	if len(key) < 5 {
		return shim.Error("clob key is empty")
	}
	rbcMessage.Data = []byte(key)

	res, err := broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
	if err != nil {
		return shim.Error(fmt.Sprintf("getAttach from orderer err: %s", err))
	}
	return shim.Success(res.Data)
}

func (t *RBCModel) getCurrentPath() string {
	execPath, err := exec.LookPath(os.Args[0])
	if err != nil {
		return ""
	}

	//    Is Symlink
	fi, err := os.Lstat(execPath)
	if err != nil {
		return ""
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		execPath, err = os.Readlink(execPath)
		if err != nil {
			return ""
		}
	}
	fp, err := filepath.Abs(execPath)
	if err != nil {
		return ""
	}
	lIndex := strings.LastIndex(fp, "/")
	return fp[:lIndex]
}
