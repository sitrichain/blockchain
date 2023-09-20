package rbctoken

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	pb "github.com/rongzer/blockchain/protos/peer"
)

type RBCStableCoin struct {
}

type GlobalCoin struct {
	TotalSupply int64
	RMBAsset    int64
}

type GlobalCoinWallet struct {
	Address           string
	GlobalCoinBalance int64
}

type GlobalCoinTransaction struct {
	TxId   string
	Time   string
	From   string
	To     string
	Amount int64
}

type GlobalCoinMortgage struct {
	WalletAddress string
	RMBAsset      int64
}

type GlobalCoinRedeem struct {
	WalletAddress string
}

func (c *RBCStableCoin) Init(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 2 {
		gc := &GlobalCoin{10000000, 1000000}
		address := wallectAddressGenerator()
		gc_wallet := &GlobalCoinWallet{address, gc.TotalSupply}
		gc_transaction := &GlobalCoinTransaction{"", "", "", address, gc.TotalSupply}
		gcBytes, _ := jsoniter.Marshal(gc)
		gc_walltBytes, _ := jsoniter.Marshal(gc_wallet)
		gc_transactionBytes, _ := jsoniter.Marshal(gc_transaction)
		stub.PutState("gc", gcBytes)
		stub.PutState("gc_wallet", gc_walltBytes)
		stub.PutState("gc_transaction", gc_transactionBytes)
	}

	return shim.Success(nil)
}

func (c *RBCStableCoin) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 2 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments Expecting %d", len(args)))
	}

	f := string(args[1])
	switch f {
	case "mortgage":
		mBytes := args[2]
		m := GlobalCoinMortgage{}
		err := jsoniter.Unmarshal(mBytes, &m)
		if err != nil {
			return shim.Error(err.Error())
		}

		return c.mortgage(stub, m)

	case "redeem":
		//rBytes := args[2]

	}

	return shim.Success(nil)
}

func getRandomBytes(len int) ([]byte, error) {
	key := make([]byte, len)

	// TODO: rand could fill less bytes then len
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func generateHash() string {
	bb := bytes.Buffer{}

	rb, _ := getRandomBytes(36)
	bb.Write(rb)

	timeBytes, _ := time.Now().MarshalBinary()
	bb.Write(timeBytes)

	sb := util.WriteAndSumSha256(bb.Bytes())

	hashCode := hex.EncodeToString(sb)
	return hashCode
}

func wallectAddressGenerator() string {
	return generateHash()
}

func (c *RBCStableCoin) mortgage(stub shim.ChaincodeStubInterface, m GlobalCoinMortgage) pb.Response {
	txId := generateHash()
	t := time.Now().String()
	gc := &GlobalCoin{}
	gcB, _ := stub.GetState("gc")
	jsoniter.Unmarshal(gcB, gc)

	gc.RMBAsset += m.RMBAsset
	gc.TotalSupply += m.RMBAsset
	walletBytes, _ := stub.GetState(m.WalletAddress)
	wallet := GlobalCoinWallet{}
	jsoniter.Unmarshal(walletBytes, &wallet)
	wallet.GlobalCoinBalance += m.RMBAsset
	walletBytes, _ = jsoniter.Marshal(wallet)
	stub.PutState(wallet.Address, walletBytes)

	gc_wallectBytes, _ := stub.GetState("gc_wallet")
	gc_wallet := &GlobalCoinWallet{}
	jsoniter.Unmarshal(gc_wallectBytes, &gc_wallet)

	transaction := GlobalCoinTransaction{txId, t, m.WalletAddress, gc_wallet.Address, m.RMBAsset}
	tb, _ := jsoniter.Marshal(transaction)
	stub.PutState(transaction.TxId, tb)

	return shim.Error("")
}
