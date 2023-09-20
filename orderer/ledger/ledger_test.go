package ledger

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/rongzer/blockchain/common/ledger"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/tool/configtxgen/provisional"
	"github.com/stretchr/testify/assert"
)

var genesisBlock = cb.NewBlock(0, nil)

type testEnv struct {
	t        *testing.T
	location string
	manager  *Manager
}

func initialize(t *testing.T) (*testEnv, *Ledger) {
	name, err := ioutil.TempDir("", "hyperledger_fabric")
	assert.NoError(t, err, "Error creating temp dir: %s", err)

	manager := NewManager(name)
	fl, err := manager.GetOrCreate(provisional.TestChainID)
	assert.NoError(t, err, "Error GetOrCreate chain")

	assert.NoError(t, fl.Append(genesisBlock))
	return &testEnv{location: name, t: t, manager: manager}, fl
}

func (tev *testEnv) tearDown() {
	tev.shutDown()
	err := os.RemoveAll(tev.location)
	if err != nil {
		tev.t.Fatalf("Error tearing down env: %s", err)
	}
}

func (tev *testEnv) shutDown() {
	tev.manager.Close()
}

type mockBlockStore struct {
	blockchainInfo             *cb.BlockchainInfo
	resultsIterator            ledger.ResultsIterator
	block                      *cb.Block
	errorTxChain               *peer.ErrorTxChain
	envelope                   *cb.Envelope
	txValidationCode           peer.TxValidationCode
	defaultError               error
	getBlockchainInfoError     error
	retrieveBlockByNumberError error
}

func (mbs *mockBlockStore) AddBlock(_ *cb.Block) error {
	return mbs.defaultError
}

func (mbs *mockBlockStore) GetBlockchainInfo() (*cb.BlockchainInfo, error) {
	return mbs.blockchainInfo, mbs.getBlockchainInfoError
}

func (mbs *mockBlockStore) RetrieveBlocks(_ uint64) (ledger.ResultsIterator, error) {
	return mbs.resultsIterator, mbs.defaultError
}

func (mbs *mockBlockStore) RetrieveBlockByHash(_ []byte) (*cb.Block, error) {
	return mbs.block, mbs.defaultError
}

func (mbs *mockBlockStore) RetrieveBlockByNumber(_ uint64) (*cb.Block, error) {
	return mbs.block, mbs.retrieveBlockByNumberError
}

func (mbs *mockBlockStore) RetrieveTxByID(_ string) (*cb.Envelope, error) {
	return mbs.envelope, mbs.defaultError
}

func (mbs *mockBlockStore) RetrieveTxByBlockNumTranNum(_ uint64, _ uint64) (*cb.Envelope, error) {
	return mbs.envelope, mbs.defaultError
}

func (mbs *mockBlockStore) RetrieveBlockByTxID(_ string) (*cb.Block, error) {
	return mbs.block, mbs.defaultError
}

func (mbs *mockBlockStore) RetrieveTxValidationCodeByTxID(_ string) (peer.TxValidationCode, error) {
	return mbs.txValidationCode, mbs.defaultError
}

func (mbs *mockBlockStore) Shutdown() {
}

func (mbs *mockBlockStore) GetAttach(_ string) string {
	return ""
}

func (mbs *mockBlockStore) RetrieveErrorTxChain(_ string) (*peer.ErrorTxChain, error) {
	return mbs.errorTxChain, mbs.defaultError
}

func (mbs *mockBlockStore) RetrieveLatestErrorTxChain() (string, error) {
	return "", mbs.defaultError
}

func TestInitialization(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()

	assert.Equal(t, uint64(1), fl.Height(), "Block height should be 1")

	block, err := fl.GetBlockByNumber(0)
	assert.NoError(t, err)
	assert.NotNil(t, block, "Error retrieving genesis block")
	assert.Equal(t, genesisBlock.Header.Hash(), block.Header.Hash(), "Block hashes did no match")
}

func TestReinitialization(t *testing.T) {
	tev, ldg := initialize(t)
	defer tev.tearDown()

	// create a block to add to the ledger
	b1, err := ldg.CreateNextBlock([]*cb.Envelope{{Payload: []byte("My Data")}})
	assert.NoError(t, err)
	// add the block to the ledger
	assert.NoError(t, ldg.Append(b1))

	_, err = tev.manager.GetOrCreate(provisional.TestChainID)
	assert.NoError(t, err, "Expected to sucessfully get test chain")

	ids, err := tev.manager.ChainIDs()
	assert.NoError(t, err, "Exptected to sucessfully get chain ids")
	assert.Equal(t, 1, len(ids), "Exptected not new chain to be created")

	// shutdown the ledger
	ldg.blockStore.Shutdown()

	// shut down the ledger provider
	tev.shutDown()

	// re-initialize the ledger provider (not the test ledger itself!)
	manager2 := NewManager(tev.location)

	// assert expected ledgers exist
	chains, err := manager2.ChainIDs()
	assert.NoError(t, err, "Exptected to sucessfully get chain ids")
	assert.Equal(t, 1, len(chains), "Should have recovered the chain")

	// get the existing test chain ledger
	ledger2, err := manager2.GetOrCreate(chains[0])
	assert.NoError(t, err, "Unexpected error: %s", err)

	block, err := ledger2.GetBlockByNumber(1)
	assert.NoError(t, err)
	assert.NotNil(t, block, "Error retrieving block 1")
	assert.Equal(t, b1.Header.Hash(), block.Header.Hash(), "Block hashes did no match")
}

func TestAddition(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()
	info, _ := fl.blockStore.GetBlockchainInfo()
	prevHash := info.CurrentBlockHash

	block, err := fl.CreateNextBlock([]*cb.Envelope{{Payload: []byte("My Data")}})
	assert.NoError(t, err)
	assert.NoError(t, fl.Append(block))
	assert.Equal(t, uint64(2), fl.Height(), "Block height should be 2")

	block, err = fl.GetBlockByNumber(1)
	assert.NoError(t, err)
	assert.NotNil(t, block, "Error retrieving genesis block")
	assert.Equal(t, prevHash, block.Header.PreviousHash, "Block hashes did no match")
}

func TestRetrieval(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()

	block, err := fl.CreateNextBlock([]*cb.Envelope{{Payload: []byte("My Data")}})
	assert.NoError(t, err)
	assert.NoError(t, fl.Append(block))
	it, num := fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Oldest{}})
	assert.Zero(t, num, "Expected genesis block iterator, but got %d", num)

	signal := it.ReadyChan()
	select {
	case <-signal:
	default:
		t.Fatalf("Should be ready for block read")
	}

	block, status, err := it.Next()
	assert.NoError(t, err)
	assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the genesis block")
	assert.Zero(t, block.Header.Number, "Expected to successfully retrieve the genesis block")

	signal = it.ReadyChan()
	select {
	case <-signal:
	default:
		t.Fatalf("Should still be ready for block read")
	}

	block, status, err = it.Next()
	assert.NoError(t, err)
	assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the second block")
	assert.Equal(
		t,
		uint64(1),
		block.Header.Number,
		"Expected to successfully retrieve the second block but got block number %d", block.Header.Number)
}

func TestBlockedRetrieval(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()
	it, num := fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: 1}}})
	if num != 1 {
		t.Fatalf("Expected block iterator at 1, but got %d", num)
	}
	assert.Equal(t, uint64(1), num, "Expected block iterator at 1, but got %d", num)

	signal := it.ReadyChan()
	select {
	case <-signal:
		t.Fatalf("Should not be ready for block read")
	default:
	}

	block, _ := fl.CreateNextBlock([]*cb.Envelope{{Payload: []byte("My Data")}})
	assert.NoError(t, fl.Append(block))
	select {
	case <-signal:
	default:
		t.Fatalf("Should now be ready for block read")
	}

	block, status, err := it.Next()
	assert.NoError(t, err)
	assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the second block")
	assert.Equal(
		t,
		uint64(1),
		block.Header.Number,
		"Expected to successfully retrieve the second block but got block number %d", block.Header.Number)

	go func() {
		block, _ := fl.CreateNextBlock([]*cb.Envelope{{Payload: []byte("My Data")}})
		assert.NoError(t, fl.Append(block))
	}()
	select {
	case <-it.ReadyChan():
		t.Fatalf("Should not be ready for block read")
	default:
		block, status, err = it.Next()
		assert.NoError(t, err)
		assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the third block")
		assert.Equal(t, uint64(2), block.Header.Number, "Expected to successfully retrieve the third block")
	}
}

func TestBlockstoreError(t *testing.T) {
	// Since this test only ensures failed GetBlockchainInfo
	// is properly handled. We don't bother creating fully
	// legit ledgers here (without genesis block).
	{
		fl := &Ledger{
			blockStore: &mockBlockStore{
				blockchainInfo:         nil,
				getBlockchainInfoError: fmt.Errorf("Error getting blockchain info"),
			},
			signal: make(chan struct{}),
		}
		assert.Panics(
			t,
			func() {
				fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Newest{}})
			},
			"Expected Iterator() to panic if blockstore operation fails")

		assert.Panics(
			t,
			func() { fl.Height() },
			"Expected Height() to panic if blockstore operation fails ")
	}

	{
		fl := &Ledger{
			blockStore: &mockBlockStore{
				blockchainInfo:             &cb.BlockchainInfo{Height: uint64(1)},
				getBlockchainInfoError:     nil,
				retrieveBlockByNumberError: fmt.Errorf("Error retrieving block by number"),
			},
			signal: make(chan struct{}),
		}
		it, _ := fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: 42}}})
		assert.IsType(t, &notFoundErrorIterator{}, it, "Expected Not Found Error if seek number is greater than ledger height")
		_, status, err := it.Next()
		assert.Error(t, err)
		assert.Equal(t, cb.Status_NOT_FOUND, status, "Expected not found error")
		select {
		case <-it.ReadyChan():
			it, _ = fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Oldest{}})
			_, status, err = it.Next()
			assert.Error(t, err)
			assert.Equal(t, cb.Status_SERVICE_UNAVAILABLE, status, "Expected service unavailable error")
		default:
			t.Fatalf("Should be closed readychan")
		}
	}
}
