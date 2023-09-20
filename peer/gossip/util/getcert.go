package util

import (
	"encoding/json"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/statedb/stateleveldb"
	"github.com/rongzer/blockchain/peer/scc/rbccustomer"
)

func GetCertFromDb(member, chainID string) (string, error) {
	vdbProvider := stateleveldb.NewVersionedDBProvider()
	dbHandler, err := vdbProvider.GetDBHandle(chainID)
	if err != nil {
		return "", err
	}
	customerId, err := dbHandler.GetState("rbccustomer", "__RBC_IDX_"+member)
	if err != nil || customerId == nil || len(customerId.Value) < 1 {
		return "", err
	}
	customerEntity, err := dbHandler.GetState("rbccustomer", string(customerId.Value))
	if err != nil || customerEntity == nil || len(customerEntity.Value) < 1 {
		return "", err
	}
	existCustomer := &rbccustomer.CustomerEntity{}
	err = json.Unmarshal(customerEntity.Value, existCustomer)
	if err != nil {
		return "", err
	}
	cert := existCustomer.CustomerSignCert
	return cert, nil
}
