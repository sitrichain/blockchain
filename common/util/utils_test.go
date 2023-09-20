package util

import (
	"bytes"
	"testing"
	"time"
)

func TestUUIDGeneration(t *testing.T) {
	uuid := GenerateUUID()
	if len(uuid) != 36 {
		t.Fatalf("UUID length is not correct. Expected = 36, Got = %d", len(uuid))
	}
	uuid2 := GenerateUUID()
	if uuid == uuid2 {
		t.Fatalf("Two UUIDs are equal. This should never occur")
	}
}

func TestTimestamp(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Logf("timestamp now: %v", CreateUtcTimestamp())
		time.Sleep(200 * time.Millisecond)
	}
}

func TestGenerateHashFromSignature(t *testing.T) {
	if bytes.Compare(GenerateHashFromSignature([]byte("aCtor12")),
		GenerateHashFromSignature([]byte("aCtor12"))) != 0 {
		t.Fatalf("Expected hashes to match, but they did not match")
	}
	if bytes.Compare(GenerateHashFromSignature([]byte("aCtor12")),
		GenerateHashFromSignature([]byte("bCtor34"))) == 0 {
		t.Fatalf("Expected hashes to be different, but they match")
	}
}

func TestMetadataSignatureBytesNormal(t *testing.T) {
	first := []byte("first")
	second := []byte("second")
	third := []byte("third")

	result := ConcatenateBytes(first, second, third)
	expected := []byte("firstsecondthird")
	if !bytes.Equal(result, expected) {
		t.Errorf("Did not concatenate bytes correctly, expected %s, got %s", expected, result)
	}
}

func TestMetadataSignatureBytesNil(t *testing.T) {
	first := []byte("first")
	second := []byte(nil)
	third := []byte("third")

	result := ConcatenateBytes(first, second, third)
	expected := []byte("firstthird")
	if !bytes.Equal(result, expected) {
		t.Errorf("Did not concatenate bytes correctly, expected %s, got %s", expected, result)
	}
}
