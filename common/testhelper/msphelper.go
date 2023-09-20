package testhelper

import (
	"path/filepath"

	m "github.com/rongzer/blockchain/common/msp"
	"github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/peer/config"
	"github.com/rongzer/blockchain/protos/msp"
)

type noopmsp struct {
}

// NewNoopMsp returns a no-op implementation of the MSP inteface
func NewNoopMsp() m.MSP {
	return &noopmsp{}
}

func (msp *noopmsp) Setup(*msp.MSPConfig) error {
	return nil
}

func (msp *noopmsp) GetType() m.ProviderType {
	return 0
}

func (msp *noopmsp) GetIdentifier() (string, error) {
	return "NOOP", nil
}

func (msp *noopmsp) GetSigningIdentity(_ *m.IdentityIdentifier) (m.SigningIdentity, error) {
	id, _ := newNoopSigningIdentity()
	return id, nil
}

func (msp *noopmsp) GetDefaultSigningIdentity() (m.SigningIdentity, error) {
	id, _ := newNoopSigningIdentity()
	return id, nil
}

// GetRootCerts returns the root certificates for this MSP
func (msp *noopmsp) GetRootCerts() []m.Identity {
	return nil
}

// GetIntermediateCerts returns the intermediate root certificates for this MSP
func (msp *noopmsp) GetIntermediateCerts() []m.Identity {
	return nil
}

// GetTLSRootCerts returns the root certificates for this MSP
func (msp *noopmsp) GetTLSRootCerts() [][]byte {
	return nil
}

// GetTLSIntermediateCerts returns the intermediate root certificates for this MSP
func (msp *noopmsp) GetTLSIntermediateCerts() [][]byte {
	return nil
}

func (msp *noopmsp) DeserializeIdentity(_ []byte) (m.Identity, error) {
	id, _ := newNoopIdentity()
	return id, nil
}

func (msp *noopmsp) Validate(_ m.Identity) error {
	return nil
}

func (msp *noopmsp) SatisfiesPrincipal(_ m.Identity, _ *msp.MSPPrincipal) error {
	return nil
}

type noopidentity struct {
}

func newNoopIdentity() (m.Identity, error) {
	return &noopidentity{}, nil
}

func (id *noopidentity) SatisfiesPrincipal(*msp.MSPPrincipal) error {
	return nil
}

func (id *noopidentity) GetIdentifier() *m.IdentityIdentifier {
	return &m.IdentityIdentifier{Mspid: "NOOP", Id: "Bob"}
}

func (id *noopidentity) GetMSPIdentifier() string {
	return "MSPID"
}

func (id *noopidentity) Validate() error {
	return nil
}

func (id *noopidentity) GetOrganizationalUnits() []*m.OUIdentifier {
	return nil
}

func (id *noopidentity) Verify(_ []byte, _ []byte) error {
	return nil
}

func (id *noopidentity) Serialize() ([]byte, error) {
	return []byte("cert"), nil
}

type noopsigningidentity struct {
	noopidentity
}

func newNoopSigningIdentity() (m.SigningIdentity, error) {
	return &noopsigningidentity{}, nil
}

func (id *noopsigningidentity) Sign(_ []byte) ([]byte, error) {
	return []byte("signature"), nil
}

func (id *noopsigningidentity) GetPublicVersion() m.Identity {
	return id
}

// LoadTestMSPSetup sets up the local MSP
// and a chain MSP for the default chain
func LoadMSPSetupForTesting() error {
	dir, err := config.GetDevConfigDir()
	if err != nil {
		return err
	}
	conf, err := m.GetLocalMspConfig(filepath.Join(dir, "msp"), nil, "DEFAULT")
	if err != nil {
		return err
	}

	err = mgmt.GetLocalMSP().Setup(conf)
	if err != nil {
		return err
	}

	err = mgmt.GetManagerForChain("testchainid").Setup([]m.MSP{mgmt.GetLocalMSP()})
	if err != nil {
		return err
	}

	return nil
}
