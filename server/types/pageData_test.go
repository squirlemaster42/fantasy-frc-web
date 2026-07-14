package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPageData(t *testing.T) {
	pageData := NewPageData(42, "test", true)

	assert.NotNil(t, pageData)
	assert.Equal(t, 42, pageData.DraftId)
	assert.Equal(t, "test", pageData.DraftName)
	assert.True(t, pageData.IsOwner)
}

func TestNewPageData_NotOwner(t *testing.T) {
	pageData := NewPageData(7, "test2", false)

	assert.NotNil(t, pageData)
	assert.Equal(t, 7, pageData.DraftId)
	assert.Equal(t, "test2", pageData.DraftName)
	assert.False(t, pageData.IsOwner)
}
