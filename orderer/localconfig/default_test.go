package localconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const DummyPath = "/dummy/path"

func TestKafkaTLSConfig(t *testing.T) {
	testCases := []struct {
		name        string
		tls         TLS
		shouldPanic bool
	}{
		{"Disabled", TLS{Enabled: false}, false},
		{"EnabledNoPrivateKey", TLS{Enabled: true, Certificate: "public.key"}, true},
		{"EnabledNoPublicKey", TLS{Enabled: true, PrivateKey: "private.key"}, true},
		{"EnabledNoTrustedRoots", TLS{Enabled: true, PrivateKey: "private.key", Certificate: "public.key"}, true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uconf := &TopLevel{Kafka: Kafka{TLS: tc.tls}}
			if tc.shouldPanic {
				assert.Error(t, uconf.setMissingValueToDefault(DummyPath), "should error")
			} else {
				assert.NoError(t, uconf.setMissingValueToDefault(DummyPath), "should not error")
			}
		})
	}
}

func TestProfileConfig(t *testing.T) {
	uconf := &TopLevel{General: General{Profile: Profile{Enabled: true}}}
	assert.NoError(t, uconf.setMissingValueToDefault(DummyPath))
	assert.Equal(t, defaults.General.Profile.Address, uconf.General.Profile.Address, "Expected profile address to be filled with default value")
}
