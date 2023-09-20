package util

type DBHandle interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte, sync bool) error
	Delete(key []byte, sync bool) error
	WriteBatch(batch *UpdateBatch, sync bool) error
	GetIterator(startKey []byte, endKey []byte) Iterator
}

type Provider interface {
	GetDBHandle(dbName string) DBHandle
	Close() error
}

type Iterator interface {
	Next() bool
	Key() []byte
	Value() []byte
	Release()
	SeekToFirst()
	Valid() bool
}

type DbUpdateBatch interface {
	Put(key, value []byte)
	Delete(key []byte)
}

type IdStoreDb interface {
	Open() error
	Close() error
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte, sync bool) error
	Delete(key []byte, sync bool) error
	WriteBatch(batch DbUpdateBatch, sync bool) error
	GetIterator(startKey []byte, endKey []byte) Iterator
}

type UpdateBatch struct {
	KVS map[string][]byte
}

func NewUpdateBatch() *UpdateBatch {
	return &UpdateBatch{KVS: make(map[string][]byte)}
}

func (batch *UpdateBatch) Get(key string) []byte {
	return batch.KVS[key]
}

func (batch *UpdateBatch) Put(key []byte, value []byte) {
	batch.KVS[string(key)] = value
}

func (batch *UpdateBatch) Delete(key []byte) {
	batch.KVS[string(key)] = nil
}

func (batch *UpdateBatch) GetKVS() map[string][]byte {
	return batch.KVS
}
