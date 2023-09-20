package ledger

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/rongzer/blockchain/common/ledger/blkstorage"
	"github.com/rongzer/blockchain/tool/configtxgen/provisional"
	"github.com/stretchr/testify/assert"
)

type mockBlockStoreProvider struct {
	blockstore blkstorage.BlockStore
	exists     bool
	list       []string
	error      error
}

func (mbsp *mockBlockStoreProvider) CreateBlockStore(_ string) (blkstorage.BlockStore, error) {
	return mbsp.blockstore, mbsp.error
}

func (mbsp *mockBlockStoreProvider) OpenBlockStore(_ string) (blkstorage.BlockStore, error) {
	return mbsp.blockstore, mbsp.error
}

func (mbsp *mockBlockStoreProvider) Exists(_ string) (bool, error) {
	return mbsp.exists, mbsp.error
}

func (mbsp *mockBlockStoreProvider) List() ([]string, error) {
	return mbsp.list, mbsp.error
}

func (mbsp *mockBlockStoreProvider) Close() {
}

func TestManager_Error(t *testing.T) {
	manager := &Manager{
		blkstorageProvider: &mockBlockStoreProvider{error: fmt.Errorf("blockstorage provider error")},
	}
	_, err := manager.ChainIDs()
	assert.Error(t, err, "Expected ChainIDs to error if storage provider cannot list chain IDs")

	_, err = manager.GetOrCreate("foo")
	assert.Error(t, err, "Expected GetOrCreate to return error if blockstorage provider cannot open")
	assert.Empty(t, manager.ledgers, "Expected no new ledger is created")
}

func TestManager_MultiReinitialization(t *testing.T) {
	dir, err := ioutil.TempDir("", "rbc")
	assert.NoError(t, err, "Error creating temp dir: %s", err)

	manager := NewManager(dir)
	_, err = manager.GetOrCreate(provisional.TestChainID)
	assert.NoError(t, err, "Error GetOrCreate chain")

	ids, err := manager.ChainIDs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids), "Expected 1 chain")
	manager.Close()

	manager = NewManager(dir)
	_, err = manager.GetOrCreate("foo")
	assert.NoError(t, err, "Error creating chain")
	l, err := manager.GetOrCreate("foo")
	assert.NoError(t, err, "Error getting chain")
	assert.NotNil(t, l)

	ids, err = manager.ChainIDs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ids), "Expected chain to be recovered")
	manager.Close()

	manager = NewManager(dir)
	_, err = manager.GetOrCreate("bar")
	assert.NoError(t, err, "Error creating chain")
	ids, err = manager.ChainIDs()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(ids), "Expected chain to be recovered")
	manager.Close()
}
