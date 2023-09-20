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

// Package shim provides APIs for the chaincode to access its state
package shim

import (
	"encoding/base64"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/rongzer/blockchain/protos/common"
)

// Customer data entity
type PageList struct {
	Cpno string        `json:"cpno"` //当前页数,
	Rnum string        `json:"rnum"` //总记录数
	List []interface{} `json:"list"` //列表对象
}

//// yu 添加排序接口
//func (p *PageList) Len() int {
//	return len(p.List)
//}
//
//func (p *PageList) Swap(i, j int) {
//	p.List[i], p.List[j] = p.List[j], p.List[i]
//}
//
//func (p *PageList) Less(i, j int) bool {
//	tab1, _ := p.List[i].(rbcmodel.RBCTable)
//	tab2, _ := p.List[i].(rbcmodel.RBCTable)
//
//	return tab1.TableName > tab2.TableName
//}

// HistoryEntity
type HistoryEntity struct {
	TxId   string `json:"txId"`   //交易id,
	TxTime string `json:"txTime"` //交易时间
	Value  string `json:"value"`  //值
}

const prefix = "__RLIST_"

func (stub *ChaincodeStub) Add(rlistName string, id string) {
	stub.seq = stub.seq + 1
	stub.PutState(prefix+"ADD:"+rlistName+","+strconv.Itoa(stub.seq), []byte(id))
}

func (stub *ChaincodeStub) AddByIndex(rlistName string, nIndex int, id string) {
	stub.seq = stub.seq + 1
	stub.PutState(prefix+"ADD:"+rlistName+","+strconv.Itoa(stub.seq), []byte(strconv.Itoa(nIndex)+","+id))
}

func (stub *ChaincodeStub) Remove(rListName string, id string) {
	stub.seq = stub.seq + 1
	stub.PutState(prefix+"DEL:"+rListName+","+strconv.Itoa(stub.seq), []byte(id))
}

func (stub *ChaincodeStub) Size(rListName string) int {
	sizeBytes, err := stub.GetState(prefix + "LEN:" + rListName)
	if err != nil {
		return 0
	}
	nSize, err := strconv.Atoi(string(sizeBytes))
	if err != nil {
		nSize = 0
	}
	return nSize
}

func (stub *ChaincodeStub) Get(rListName string, nIndex int) string {
	sizeBytes, err := stub.GetState(prefix + "GET:" + rListName + "," + strconv.Itoa(nIndex))
	if err != nil {
		return ""
	}
	return string(sizeBytes)
}

func (stub *ChaincodeStub) IndexOf(rListName string, id string) int {
	returnBytes, err := stub.GetState(prefix + "IDX:" + rListName + "," + id)
	if err != nil {
		return -1
	}
	nReturn, err := strconv.Atoi(string(returnBytes))
	if err != nil {
		nReturn = -1
	}
	return nReturn
}

func (stub *ChaincodeStub) GetHeightById(rListName string, id string) int {
	returnBytes, err := stub.GetState(prefix + "HGT:" + rListName + "," + id)
	if err != nil {
		return -1
	}
	nReturn, err := strconv.Atoi(string(returnBytes))
	if err != nil {
		nReturn = -1
	}
	return nReturn
}

func (stub *ChaincodeStub) GetIdsByHeight(rListName string, height int) ([]string, int) {
	returnBytes, err := stub.GetState(prefix + "IDH:" + rListName + "," + strconv.Itoa(height))
	if err != nil {
		return nil, -1
	}

	rListHeight := &common.RListHeight{}
	if err = rListHeight.Unmarshal(returnBytes); err != nil {
		return nil, -1
	}
	return rListHeight.Ids, int(rListHeight.NextHeight)
}

//取状态值的变更历史
func (stub *ChaincodeStub) QueryStateHistory(idKey string) []byte {
	if len(idKey) < 1 {
		return []byte("")
	}
	keysIter, err := stub.GetHistoryForKey(idKey)
	if err != nil {
		return []byte("")
	}
	defer keysIter.Close()

	listValue := make([]interface{}, 0)
	pageList := &PageList{Cpno: "0", List: listValue}

	nSize := 0
	for keysIter.HasNext() {
		response, iterErr := keysIter.Next()

		if iterErr != nil {
			return []byte("")
		}
		tm := time.Unix(response.Timestamp.Seconds, 0)
		txTimeStr := tm.Format("2006-01-02 15:04:05")
		historyEntity := &HistoryEntity{TxId: response.TxId, TxTime: txTimeStr}
		historyEntity.Value = base64.StdEncoding.EncodeToString(response.Value)
		pageList.List = append(pageList.List, historyEntity)

		nSize++
	}
	pageList.Rnum = strconv.Itoa(nSize)
	pageListBuf, _ := jsoniter.Marshal(pageList)
	return pageListBuf
}

//计数器前缀
const prefix_rcounter = "__RCOUNTER_"

//取计数器的当前值
func (stub *ChaincodeStub) GetValue(rCounterName string) int64 {
	returnBytes, err := stub.GetState(prefix_rcounter + rCounterName)
	lValue := int64(0)
	if err == nil {
		lValue, err = strconv.ParseInt(string(returnBytes), 10, 64)
		if err != nil {
			lValue = int64(0)
		}
	}
	return lValue
}

//设置计数器的值
func (stub *ChaincodeStub) SetValue(rCounterName string, lValue int64) {
	stub.PutState(prefix_rcounter+rCounterName, []byte(strconv.FormatInt(lValue, 10)))
}

//计数器的加法
func (stub *ChaincodeStub) AddValue(rCounterName string, lValue int64) {
	stub.seq = stub.seq + 1

	stub.PutState("__CAL_ADD:"+prefix_rcounter+rCounterName+","+strconv.Itoa(stub.seq), []byte(strconv.FormatInt(lValue, 10)))

}

//计数器的乘法
func (stub *ChaincodeStub) MulValue(rCounterName string, lValue int64) {
	stub.seq = stub.seq + 1

	stub.PutState("__CAL_MUL:"+prefix_rcounter+rCounterName+","+strconv.Itoa(stub.seq), []byte(strconv.FormatInt(lValue, 10)))
}

//计数器的当前动态值字符串
func (stub *ChaincodeStub) GetCurrentValue(rCounterName string) string {
	return "##RCOUNT##" + prefix_rcounter + rCounterName + "##RCOUNT##"
}
