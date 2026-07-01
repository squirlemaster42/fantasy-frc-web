package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPageData(t *testing.T) {
	pageData := NewPageData(42, true)

	assert.NotNil(t, pageData)
	assert.Equal(t, 42, pageData.DraftId)
	assert.True(t, pageData.IsOwner)
}

func TestNewPageData_NotOwner(t *testing.T) {
	pageData := NewPageData(7, false)

	assert.NotNil(t, pageData)
	assert.Equal(t, 7, pageData.DraftId)
	assert.False(t, pageData.IsOwner)
}
