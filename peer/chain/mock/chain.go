package mock

import (
	"github.com/rongzer/blockchain/common/testhelper"
	mockconfigtx "github.com/rongzer/blockchain/common/testhelper/configtx"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/ledger/ledgermgmt"
)

//MockInitialize resets chains for test env
func MockInitialize() {
	ledgermgmt.InitializeTestEnv()

	chain.ChainsInit()
}

var mockMSPIDGetter func(string) []string

func MockSetMSPIDGetter(mspIDGetter func(string) []string) {
	mockMSPIDGetter = mspIDGetter
}

func MockCreateChain(cid string) error {
	var ldger ledger.PeerLedger
	var err error

	if ldger = chain.GetLedger(cid); ldger == nil {
		gb, _ := testhelper.MakeGenesisBlock(cid)
		if ldger, err = ledgermgmt.CreateLedger(gb); err != nil {
			return err
		}
	}

	// Here we need to mock also the policy manager
	// in order for the ACL to be checked
	initializer := mockconfigtx.Initializer{
		Resources: mockconfigtx.Resources{
			PolicyManagerVal: &testhelper.Manager{
				Policy: &testhelper.Policy{},
			},
		},
		PolicyProposerVal: &mockconfigtx.PolicyProposer{
			Transactional: mockconfigtx.Transactional{},
		},
		ValueProposerVal: &mockconfigtx.ValueProposer{
			Transactional: mockconfigtx.Transactional{},
		},
	}
	manager := &mockconfigtx.Manager{
		Initializer: initializer,
	}

	chain.AddChain(cid, ldger, manager)

	return nil
}
