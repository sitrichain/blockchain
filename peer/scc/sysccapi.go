package scc

import (
	"fmt"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/ledger"
	"strings"

	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/common/ccprovider"
	"github.com/rongzer/blockchain/peer/container/inproccontroller"
	pb "github.com/rongzer/blockchain/protos/peer"
	"golang.org/x/net/context"
)

// SystemChaincode defines the metadata needed to initialize system chaincode
// when the blockchain comes up. SystemChaincodes are installed by adding an
// entry in importsysccs.go
type SystemChaincode struct {
	//Unique name of the system chaincode
	Name string

	//Path to the system chaincode; currently not used
	Path string

	//InitArgs initialization arguments to startup the system chaincode
	InitArgs [][]byte

	// Chaincode is the actual chaincode object
	Chaincode shim.Chaincode

	// InvokableExternal keeps track of whether
	// this system chaincode can be invoked
	// through a proposal sent to this peer
	InvokableExternal bool

	// InvokableCC2CC keeps track of whether
	// this system chaincode can be invoked
	// by way of a chaincode-to-chaincode
	// invocation
	InvokableCC2CC bool

	// Enabled a convenient switch to enable/disable system chaincode without
	// having to remove entry from importsysccs.go
	Enabled bool
}

// RegisterSysCC registers the given system chaincode with the peer
func RegisterSysCC(syscc *SystemChaincode) error {
	if !syscc.Enabled || !isWhitelisted(syscc) {
		log.Logger.Info(fmt.Sprintf("system chaincode (%s,%s,%t) disabled", syscc.Name, syscc.Path, syscc.Enabled))
		return nil
	}

	err := inproccontroller.Register(syscc.Path, syscc.Chaincode)
	if err != nil {
		//if the type is registered, the instance may not be... keep going
		if _, ok := err.(inproccontroller.SysCCRegisteredErr); !ok {
			errStr := fmt.Sprintf("could not register (%s,%v): %s", syscc.Path, syscc, err)
			log.Logger.Error(errStr)
			return fmt.Errorf(errStr)
		}
	}

	log.Logger.Debugf("system chaincode %s(%s) registered", syscc.Name, syscc.Path)
	return err
}

// deploySysCC deploys the given system chaincode on a chain
func deploySysCC(chainID string, syscc *SystemChaincode) error {
	if !syscc.Enabled || !isWhitelisted(syscc) {
		log.Logger.Info(fmt.Sprintf("system chaincode (%s,%s) disabled", syscc.Name, syscc.Path))
		return nil
	}

	var err error

	ccprov := ccprovider.GetChaincodeProvider()

	ctxt := context.Background()
	if chainID != "" {
		// 如果是raft共识模式下，不能从peer/chain包的chains全局变量中获取lgr，因为raft共识模式下，peer模块不再创建链和账本（orderer模块已经创建过了）
		var lgr ledger.PeerLedger
		lgr = chain.GetLedger(chainID)
		if lgr == nil {
			panic(fmt.Sprintf("syschaincode %s start up failure - unexpected nil ledger for channel %s", syscc.Name, chainID))
		}

		_, err := ccprov.GetContext(lgr)
		if err != nil {
			return err
		}

		defer ccprov.ReleaseContext()
	}

	chaincodeID := &pb.ChaincodeID{Path: syscc.Path, Name: syscc.Name}
	spec := &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: &pb.ChaincodeInput{Args: syscc.InitArgs}}

	// First build and get the deployment spec
	chaincodeDeploymentSpec, err := buildSysCC(ctxt, spec)

	if err != nil {
		log.Logger.Error(fmt.Sprintf("Error deploying chaincode spec: %v\n\n error: %s", spec, err))
		return err
	}

	txid := util.GenerateUUID()

	version := util.GetSysCCVersion()

	cccid := ccprov.GetCCContext(chainID, chaincodeDeploymentSpec.ChaincodeSpec.ChaincodeId.Name, version, txid, true, nil, nil)

	_, _, err = ccprov.ExecuteWithErrorFilter(ctxt, cccid, chaincodeDeploymentSpec)

	log.Logger.Infof("system chaincode %s of %s deployed", syscc.Name, chainID)

	return err
}

// buildLocal builds a given chaincode code
func buildSysCC(_ context.Context, spec *pb.ChaincodeSpec) (*pb.ChaincodeDeploymentSpec, error) {
	var codePackageBytes []byte
	chaincodeDeploymentSpec := &pb.ChaincodeDeploymentSpec{ExecEnv: pb.ChaincodeDeploymentSpec_SYSTEM, ChaincodeSpec: spec, CodePackage: codePackageBytes}
	return chaincodeDeploymentSpec, nil
}

func isWhitelisted(syscc *SystemChaincode) bool {
	// 增加系统chaincode的白名单
	if strings.HasPrefix(syscc.Name, "rbc") || syscc.Name == "simple" {
		return true
	}

	for i := range conf.V.Peer.Chaincode.WhiteList {
		if syscc.Name == conf.V.Peer.Chaincode.WhiteList[i] {
			return true
		}
	}

	return false
}
