package executor

import (
	"context"
	"fmt"
	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/peer/chaincode"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/common/ccprovider"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/scc"
	"github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

type Executor struct {

}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) disableJavaCCInst(cid *peer.ChaincodeID, cis *peer.ChaincodeInvocationSpec) error {
	//if not lscc we don't care
	if cid.Name != "lscc" {
		return nil
	}

	//non-nil spec ? leave it to callers to handle error if this is an error
	if cis.ChaincodeSpec == nil || cis.ChaincodeSpec.Input == nil {
		return nil
	}

	//should at least have a command arg, leave it to callers if this is an error
	if len(cis.ChaincodeSpec.Input.Args) < 1 {
		return nil
	}

	var argNo int
	switch string(cis.ChaincodeSpec.Input.Args[0]) {
	case "install":
		argNo = 1
	case "deploy", "upgrade":
		argNo = 2
	default:
		//what else can it be ? leave it caller to handle it if error
		return nil
	}

	//the inner dep spec will contain the type
	cds, err := utils.GetChaincodeDeploymentSpec(cis.ChaincodeSpec.Input.Args[argNo])
	if err != nil {
		return err
	}

	//finally, if JAVA error out
	if cds.ChaincodeSpec.Type == peer.ChaincodeSpec_JAVA {
		return nil
	}

	//not a java install, instantiate or upgrade op
	return nil
}

func (e *Executor) getCDSFromLSCC(ctx context.Context, chainID string, txid string, signedProp *peer.SignedProposal, prop *peer.Proposal,
	chaincodeID string, txsim ledger.TxSimulator) (*ccprovider.ChaincodeData, error) {
	ctxt := ctx
	if txsim != nil {
		ctxt = context.WithValue(ctx, chaincode.TXSimulatorKey, txsim)
	}

	return chaincode.GetChaincodeDataFromLSCC(ctxt, txid, signedProp, prop, chainID, chaincodeID)
}

func (e *Executor) callChaincode(ctxt context.Context, chainID string, version string, txid string, signedProp *peer.SignedProposal, prop *peer.Proposal,
	cis *peer.ChaincodeInvocationSpec, cid *peer.ChaincodeID, txsim ledger.TxSimulator) (*peer.Response, *peer.ChaincodeEvent, error) {

	var err error
	var res *peer.Response
	var ccevent *peer.ChaincodeEvent

	if txsim != nil {
		ctxt = context.WithValue(ctxt, chaincode.TXSimulatorKey, txsim)
	}

	//is this a system chaincode
	isscc := scc.IsSysCC(cid.Name)
	cccid := ccprovider.NewCCContext(chainID, cid.Name, version, txid, isscc, signedProp, prop)

	res, ccevent, err = chaincode.ExecuteChaincode(ctxt, cccid, cis.ChaincodeSpec.Input.Args)

	if err != nil {
		return nil, nil, err
	}

	//per doc anything < 400 can be sent as TX.
	//blockchain errors will always be >= 400 (ie, unambiguous errors )
	//"lscc" will respond with status 200 or 500 (ie, unambiguous OK or ERROR)
	if res.Status >= shim.ERRORTHRESHOLD {
		return res, nil, nil
	}

	//----- BEGIN -  SECTION THAT MAY NEED TO BE DONE IN LSCC ------
	//if this a call to deploy a chaincode, We need a mechanism
	//to pass TxSimulator into LSCC. Till that is worked out this
	//special code does the actual deploy, upgrade here so as to collect
	//all state under one TxSimulator
	//
	//NOTE that if there's an error all simulation, including the chaincode
	//table changes in lscc will be thrown away
	if cid.Name == "lscc" && len(cis.ChaincodeSpec.Input.Args) >= 3 && (string(cis.ChaincodeSpec.Input.Args[0]) == "deploy" || string(cis.ChaincodeSpec.Input.Args[0]) == "upgrade") {
		var cds *peer.ChaincodeDeploymentSpec
		cds, err = utils.GetChaincodeDeploymentSpec(cis.ChaincodeSpec.Input.Args[2])
		if err != nil {
			return nil, nil, err
		}

		//this should not be a system chaincode
		if scc.IsSysCC(cds.ChaincodeSpec.ChaincodeId.Name) {
			return nil, nil, fmt.Errorf("attempting to deploy a system chaincode %s/%s", cds.ChaincodeSpec.ChaincodeId.Name, chainID)
		}

		cccid = ccprovider.NewCCContext(chainID, cds.ChaincodeSpec.ChaincodeId.Name, cds.ChaincodeSpec.ChaincodeId.Version, txid, false, signedProp, prop)

		_, _, err = chaincode.Execute(ctxt, cccid, cds)
		if err != nil {
			return nil, nil, fmt.Errorf("%s", err)
		}
	}
	//----- END -------

	return res, ccevent, err
}

func (e *Executor) SimulateProposal(ctx context.Context, chainID string, txid string, signedProp *peer.SignedProposal, prop *peer.Proposal,
	cid *peer.ChaincodeID, txsim ledger.TxSimulator) (*ccprovider.ChaincodeData, *peer.Response, []byte, *peer.ChaincodeEvent, error)   {

	//we do expect the payload to be a ChaincodeInvocationSpec
	//if we are supporting other payloads in future, this be glaringly point
	//as something that should change
	cis, err := utils.GetChaincodeInvocationSpec(prop)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	//disable Java install,instantiate,upgrade for now
	if err = e.disableJavaCCInst(cid, cis); err != nil {
		return nil, nil, nil, nil, err
	}

	var cdLedger *ccprovider.ChaincodeData
	var version string

	if !scc.IsSysCC(cid.Name) {
		cdLedger, err = e.getCDSFromLSCC(ctx, chainID, txid, signedProp, prop, cid.Name, txsim)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("%s - make sure the chaincode %s has been successfully instantiated and try again", err, cid.Name)
		}
		version = cdLedger.Version

		err = ccprovider.CheckInsantiationPolicy(cid.Name, version, cdLedger)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	} else {
		version = util.GetSysCCVersion()
	}

	//---3. execute the proposal and get simulation results
	var simResult []byte
	var res *peer.Response
	var ccevent *peer.ChaincodeEvent
	res, ccevent, err = e.callChaincode(ctx, chainID, version, txid, signedProp, prop, cis, cid, txsim)
	if err != nil {

		return nil, nil, nil, nil, err
	}

	if txsim != nil {
		if simResult, err = txsim.GetTxSimulationResults(); err != nil {
			return nil, nil, nil, nil, err
		}
	}

	return cdLedger, res, simResult, ccevent, nil
}

//endorse the proposal by calling the ESCC
func (e *Executor) EndorseProposal(ctx context.Context, chainID string, txid string, signedProp *peer.SignedProposal, proposal *peer.Proposal,
	response *peer.Response, simRes []byte, event *peer.ChaincodeEvent, visibility []byte, ccid *peer.ChaincodeID, txsim ledger.TxSimulator,
	cd *ccprovider.ChaincodeData) (*peer.ProposalResponse, error) {

	isSysCC := cd == nil
	// 1) extract the name of the escc that is requested to endorse this chaincode
	var escc string
	//ie, not "lscc" or system chaincodes
	if isSysCC {
		// FIXME: getCDSFromLSCC seems to fail for lscc - not sure this is expected?
		// TODO: who should endorse a call to LSCC?
		escc = "escc"
	} else {
		escc = cd.Escc
		if escc == "" { // this should never happen, LSCC always fills this field
			panic("No ESCC specified in ChaincodeData")
		}
	}

	// marshalling event bytes
	var err error
	var eventBytes []byte
	if event != nil {
		eventBytes, err = utils.GetBytesChaincodeEvent(event)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event bytes - %s", err)
		}
	}

	resBytes, err := utils.GetBytesResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response bytes - %s", err)
	}

	// set version of executing chaincode
	if isSysCC {
		// if we want to allow mixed blockchain levels we should
		// set syscc version to ""
		ccid.Version = util.GetSysCCVersion()
	} else {
		ccid.Version = cd.Version
	}

	ccidBytes, err := utils.Marshal(ccid)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ChaincodeID - %s", err)
	}

	// 3) call the ESCC we've identified
	// arguments:
	// args[0] - function name (not used now)
	// args[1] - serialized Header object
	// args[2] - serialized ChaincodeProposalPayload object
	// args[3] - ChaincodeID of executing chaincode
	// args[4] - result of executing chaincode
	// args[5] - binary blob of simulation results
	// args[6] - serialized events
	// args[7] - payloadVisibility
	args := [][]byte{[]byte(""), proposal.Header, proposal.Payload, ccidBytes, resBytes, simRes, eventBytes, visibility}
	version := util.GetSysCCVersion()
	ecccis := &peer.ChaincodeInvocationSpec{ChaincodeSpec: &peer.ChaincodeSpec{Type: peer.ChaincodeSpec_GOLANG, ChaincodeId: &peer.ChaincodeID{Name: escc},
		Input: &peer.ChaincodeInput{Args: args}}}
	res, _, err := e.callChaincode(ctx, chainID, version, txid, signedProp, proposal, ecccis, &peer.ChaincodeID{Name: escc}, txsim)
	if err != nil {
		return nil, err
	}

	if res.Status >= shim.ERRORTHRESHOLD {
		return &peer.ProposalResponse{Response: res}, nil
	}

	prBytes := res.Payload
	// Note that we do not extract any simulation results from
	// the call to ESCC. This is intentional becuse ESCC is meant
	// to endorse (i.e. sign) the simulation results of a chaincode,
	// but it can't obviously sign its own. Furthermore, ESCC runs
	// on private input (its own signing key) and so if it were to
	// produce simulationr results, they are likely to be different
	// from other ESCCs, which would stand in the way of the
	// endorsement process.

	//3 -- respond
	pResp, err := utils.GetProposalResponse(prBytes)
	if err != nil {
		return nil, err
	}

	return pResp, nil
}


