/*
Copyright IBM Corp. 2017 All Rights Reserved.

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

package ccprovider

import (
	"context"

	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/common/ccprovider"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/protos/peer"
)

// MockCcProviderFactory is a factory that returns
// mock implementations of the ccprovider.ChaincodeProvider interface
type MockCcProviderFactory struct {
}

// NewChaincodeProvider returns a mock implementation of the ccprovider.ChaincodeProvider interface
func (c *MockCcProviderFactory) NewChaincodeProvider() ccprovider.ChaincodeProvider {
	return &mockCcProviderImpl{}
}

// mockCcProviderImpl is a mock implementation of the chaincode provider
type mockCcProviderImpl struct {
}

type mockCcProviderContextImpl struct {
}

func (c *mockCcProviderImpl) LockTxsim() {
	panic("implement me")
}

func (c *mockCcProviderImpl) UnLockTxsim() {
	panic("implement me")
}

func (c *mockCcProviderImpl) GetTxsim() ledger.TxSimulator {
	panic("implement me")
}

func (c *mockCcProviderImpl) TxsimDone(_ ledger.TxSimulator) {
	panic("implement me")
}

// GetContext does nothing
func (c *mockCcProviderImpl) GetContext(_ ledger.PeerLedger) (context.Context, error) {
	return nil, nil
}

// GetCCContext does nothing
func (c *mockCcProviderImpl) GetCCContext(_, _, _, _ string, _ bool, _ *peer.SignedProposal, _ *peer.Proposal) interface{} {
	return &mockCcProviderContextImpl{}
}

// GetCCValidationInfoFromLSCC does nothing
func (c *mockCcProviderImpl) GetCCValidationInfoFromLSCC(_ context.Context, _ string, _ *peer.SignedProposal, _ *peer.Proposal, _ string, _ string) (string, []byte, error) {
	return "vscc", nil, nil
}

// ExecuteChaincode does nothing
func (c *mockCcProviderImpl) ExecuteChaincode(_ context.Context, _ interface{}, _ [][]byte) (*peer.Response, *peer.ChaincodeEvent, error) {
	return &peer.Response{Status: shim.OK}, nil, nil
}

// Execute executes the chaincode given context and spec (invocation or deploy)
func (c *mockCcProviderImpl) Execute(_ context.Context, _ interface{}, _ interface{}) (*peer.Response, *peer.ChaincodeEvent, error) {
	return nil, nil, nil
}

// ExecuteWithErrorFilder executes the chaincode given context and spec and returns payload
func (c *mockCcProviderImpl) ExecuteWithErrorFilter(_ context.Context, _ interface{}, _ interface{}) ([]byte, *peer.ChaincodeEvent, error) {
	return nil, nil, nil
}

// Stop stops the chaincode given context and deployment spec
func (c *mockCcProviderImpl) Stop(_ context.Context, _ interface{}, _ *peer.ChaincodeDeploymentSpec) error {
	return nil
}

// ReleaseContext does nothing
func (c *mockCcProviderImpl) ReleaseContext() {
}
