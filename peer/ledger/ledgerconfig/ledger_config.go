package ledgerconfig

import (
	"path/filepath"

	"github.com/rongzer/blockchain/common/conf"
)

// GetRootPath returns the filesystem path.
// All ledger related contents are expected to be stored under this path
func GetRootPath() string {
	return filepath.Join(conf.V.FileSystemPath, "ledgersData")
}

// GetLedgerProviderPath returns the filesystem path for stroing ledger ledgerProvider contents
func GetLedgerProviderPath() string {
	if conf.V.Ledger.StateDatabase == "rocksdb" {
		return filepath.Join(GetRootPath(), "ledgerProviderRocksdb")
	}
	return filepath.Join(GetRootPath(), "ledgerProviderLeveldb")
}

// GetStateLevelDBPath returns the filesystem path that is used to maintain the state level db
func GetStateLevelDBPath() string {
	return filepath.Join(GetRootPath(), "stateLeveldb")
}

// GetStateRocksDBPath returns the filesystem path that is used to maintain the state rocksdb
func GetStateRocksDBPath() string {
	return filepath.Join(GetRootPath(), "stateRocksdb")
}

// GetIndexLevelDBPath returns the filesystem path that is used to maintain the index level db
func GetIndexLevelDBPath() string {
	return filepath.Join(GetRootPath(), "indexLeveldb")
}

// GetIndexRocksDBPath returns the filesystem path that is used to maintain the index rocksdb
func GetIndexRocksDBPath() string {
	return filepath.Join(GetRootPath(), "indexRocksdb")
}

// GetRbcLevelDBPath returns the filesystem path that is used to maintain the rbc system level db
func GetRbcLevelDBPath() string {
	return filepath.Join(GetRootPath(), "rbcLeveldb")
}

// GetRbcLevelDBPath returns the filesystem path that is used to maintain the rbc system rocksdb
func GetRbcRocksDBPath() string {
	return filepath.Join(GetRootPath(), "rbcRocksdb")
}

// GetHistoryLevelDBPath returns the filesystem path that is used to maintain the history level db
func GetHistoryLevelDBPath() string {
	return filepath.Join(GetRootPath(), "historyLeveldb")
}

// GetHistoryLevelDBPath returns the filesystem path that is used to maintain the history rocksdb
func GetHistoryRocksDBPath() string {
	return filepath.Join(GetRootPath(), "historyRocksdb")
}

// GetBlockStorePath returns the filesystem path that is used for the chain block stores
func GetBlockStorePath() string {
	if conf.V.Ledger.StateDatabase == "rocksdb" {
		return filepath.Join(GetRootPath(), "chainsRocksdb")
	}
	return filepath.Join(GetRootPath(), "chains")
}

// GetMaxBlockfileSize returns maximum size of the block file
func GetMaxBlockfileSize() int {
	return 64 * 1024 * 1024
}

//IsHistoryDBEnabled exposes the historyDatabase variable
func IsHistoryDBEnabled() bool {
	return conf.V.Ledger.EnableHistoryDatabase
}

// IsQueryReadsHashingEnabled enables or disables computing of hash
// of range query results for phantom item validation
func IsQueryReadsHashingEnabled() bool {
	return true
}

// GetMaxDegreeQueryReadsHashing return the maximum degree of the merkle tree for hashes of
// of range query results for phantom item validation
// For more details - see description in kvledger/txmgmt/rwset/query_results_helper.go
func GetMaxDegreeQueryReadsHashing() uint32 {
	return 50
}
