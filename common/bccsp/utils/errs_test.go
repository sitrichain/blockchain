package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrToString(t *testing.T) {
	assert.Equal(t, ErrToString(errors.New("error")), "error")

	assert.Equal(t, ErrToString(nil), "<clean>")
}
