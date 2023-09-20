package conf

import (
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"

	"github.com/stretchr/testify/assert"
)

func TestReadString(t *testing.T) {
	assert.Equal(t, "peer", V.Role)
	os.Setenv("BLOCKCHAIN_ROLE", "orderer")
	assert.NoError(t, Initialize())
	assert.Equal(t, "orderer", V.Role)
}

func TestReadInt(t *testing.T) {
	assert.Equal(t, 2000, V.Peer.Events.BufferSize)
	os.Setenv("BLOCKCHAIN_PEER_EVENTS_BUFFERSIZE", "123456")
	assert.NoError(t, Initialize())
	assert.Equal(t, 123456, V.Peer.Events.BufferSize)
}

func TestReadBool(t *testing.T) {
	assert.Equal(t, false, V.Profile.Enabled)
	os.Setenv("BLOCKCHAIN_PROFILE_ENABLED", "true")
	assert.NoError(t, Initialize())
	assert.Equal(t, true, V.Profile.Enabled)
	os.Setenv("BLOCKCHAIN_PROFILE_ENABLED", "False")
	assert.NoError(t, Initialize())
	assert.Equal(t, false, V.Profile.Enabled)
}

func TestReadStringSlice(t *testing.T) {
	assert.Equal(t, []string{"tls/ca.crt"}, V.TLS.RootCAs)
	os.Setenv("BLOCKCHAIN_TLS_ROOTCAS", "a,b,c,d")
	assert.NoError(t, Initialize())
	assert.Equal(t, []string{"a", "b", "c", "d"}, V.TLS.RootCAs)
}

func TestReadDuration(t *testing.T) {
	assert.Equal(t, time.Second*10, V.Sealer.Kafka.Retry.NetworkTimeouts.DialTimeout)
	os.Setenv("BLOCKCHAIN_ORDERER_KAFKA_RETRY_NETWORKTIMEOUTS_DIALTIMEOUT", "6m6s")
	assert.NoError(t, Initialize())
	assert.Equal(t, time.Second*60*6+time.Second*6, V.Sealer.Kafka.Retry.NetworkTimeouts.DialTimeout)
}

func TestReadKafkaVersion(t *testing.T) {
	assert.Equal(t, sarama.V0_10_2_0, V.Sealer.Kafka.Version)
	os.Setenv("BLOCKCHAIN_ORDERER_KAFKA_VERSION", "2.3.0")
	assert.NoError(t, Initialize())
	assert.Equal(t, sarama.V2_3_0_0, V.Sealer.Kafka.Version)
}

func TestReadFloat64(t *testing.T) {
	assert.Equal(t, 10.0, V.Ledger.RocksDB.MaxBytesForLevelMultiplier)
	os.Setenv("BLOCKCHAIN_LEDGER_ROCKSDB_MAXBYTESFORLEVELMULTIPLIER", "1.23")
	assert.NoError(t, Initialize())
	assert.Equal(t, 1.23, V.Ledger.RocksDB.MaxBytesForLevelMultiplier)
}
