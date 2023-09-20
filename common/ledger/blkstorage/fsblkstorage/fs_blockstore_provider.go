package fsblkstorage

import (
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/ledger/blkstorage"
	"github.com/rongzer/blockchain/common/ledger/util"
	"github.com/rongzer/blockchain/common/ledger/util/leveldbhelper"
	"github.com/rongzer/blockchain/common/ledger/util/rocksdbhelper"
)

// FsBlockstoreProvider provides handle to block storage - this is not thread-safe
type FsBlockstoreProvider struct {
	conf            *Conf
	indexConfig     *blkstorage.IndexConfig
	leveldbProvider util.Provider
	attachProvider  util.Provider
}

// NewProvider constructs a filesystem based block store provider
func NewProvider(c *Conf, indexConfig *blkstorage.IndexConfig) blkstorage.BlockStoreProvider {
	var p, ap util.Provider
	if conf.V.Ledger.StateDatabase == "rocksdb" {
		p = rocksdbhelper.NewProvider(&rocksdbhelper.Conf{DBPath: c.getIndexDir()})
		ap = rocksdbhelper.NewProvider(&rocksdbhelper.Conf{DBPath: c.getAttachDir()})
	} else {
		p = leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: c.getIndexDir()})
		ap = leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: c.getAttachDir()})
	}
	return &FsBlockstoreProvider{c, indexConfig, p, ap}
}

// CreateBlockStore simply calls OpenBlockStore
func (p *FsBlockstoreProvider) CreateBlockStore(ledgerid string) (blkstorage.BlockStore, error) {
	return p.OpenBlockStore(ledgerid)
}

// OpenBlockStore opens a block store for given ledgerid.
// If a blockstore is not existing, this method creates one
// This method should be invoked only once for a particular ledgerid
func (p *FsBlockstoreProvider) OpenBlockStore(ledgerid string) (blkstorage.BlockStore, error) {
	indexStoreHandle := p.leveldbProvider.GetDBHandle(ledgerid)
	return newFsBlockStore(ledgerid, p.conf, p.indexConfig, indexStoreHandle), nil
}

// Exists tells whether the BlockStore with given id exists
func (p *FsBlockstoreProvider) Exists(ledgerid string) (bool, error) {
	exists, _, err := util.FileExists(p.conf.getLedgerBlockDir(ledgerid))
	return exists, err
}

// List lists the ids of the existing ledgers
func (p *FsBlockstoreProvider) List() ([]string, error) {
	return util.ListSubdirs(p.conf.getChainsDir())
}

// Close closes the FsBlockstoreProvider
func (p *FsBlockstoreProvider) Close() {
	p.leveldbProvider.Close()
}
