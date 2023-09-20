package filters

import (
	"testing"

	cb "github.com/rongzer/blockchain/protos/common"
)

func TestNoRule(t *testing.T) {
	rs := Set{}
	_, err := rs.Apply(&cb.Envelope{})
	if err == nil {
		t.Fatalf("Should have rejected")
	}
}
