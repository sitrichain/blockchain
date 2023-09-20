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

package chaincode

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/common/ccprovider"
	pb "github.com/rongzer/blockchain/protos/peer"
	"golang.org/x/net/context"
)

//create a chaincode invocation spec
func createCIS(ccname string, args [][]byte) (*pb.ChaincodeInvocationSpec, error) {
	spec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: ccname}, Input: &pb.ChaincodeInput{Args: args}}}
	return spec, nil
}

// GetCDSFromLSCC gets chaincode deployment spec from LSCC
func GetCDSFromLSCC(ctxt context.Context, txid string, signedProp *pb.SignedProposal, prop *pb.Proposal, chainID string, chaincodeID string) ([]byte, error) {
	version := util.GetSysCCVersion()
	cccid := ccprovider.NewCCContext(chainID, "lscc", version, txid, true, signedProp, prop)
	res, _, err := ExecuteChaincode(ctxt, cccid, [][]byte{[]byte("getdepspec"), []byte(chainID), []byte(chaincodeID)})
	if err != nil {
		return nil, fmt.Errorf("Execute getdepspec(%s, %s) of LSCC error: %s", chainID, chaincodeID, err)
	}
	if res.Status != shim.OK {
		return nil, fmt.Errorf("Get ChaincodeDeploymentSpec for %s/%s from LSCC error: %s", chaincodeID, chainID, res.Message)
	}

	return res.Payload, nil
}

// GetChaincodeDataFromLSCC gets chaincode data from LSCC given name
func GetChaincodeDataFromLSCC(ctxt context.Context, txid string, signedProp *pb.SignedProposal, prop *pb.Proposal, chainID string, chaincodeID string) (*ccprovider.ChaincodeData, error) {
	version := util.GetSysCCVersion()
	cccid := ccprovider.NewCCContext(chainID, "lscc", version, txid, true, signedProp, prop)
	res, _, err := ExecuteChaincode(ctxt, cccid, [][]byte{[]byte("getccdata"), []byte(chainID), []byte(chaincodeID)})
	if err == nil {
		if res.Status != shim.OK {
			return nil, fmt.Errorf("%s", res.Message)
		}
		cd := &ccprovider.ChaincodeData{}
		err = proto.Unmarshal(res.Payload, cd)
		if err != nil {
			return nil, err
		}
		return cd, nil
	}

	return nil, err
}

// ExecuteChaincode executes a given chaincode given chaincode name and arguments
func ExecuteChaincode(ctxt context.Context, cccid *ccprovider.CCContext, args [][]byte) (*pb.Response, *pb.ChaincodeEvent, error) {
	var spec *pb.ChaincodeInvocationSpec
	var err error
	var res *pb.Response
	var ccevent *pb.ChaincodeEvent

	spec, err = createCIS(cccid.Name, args)
	res, ccevent, err = Execute(ctxt, cccid, spec)
	if err != nil {
		log.Logger.Errorf("Error executing chaincode %s : %s", cccid.TxID, err)
		return nil, nil, fmt.Errorf("Error executing chaincode: %s", err)
	}

	return res, ccevent, err
}
