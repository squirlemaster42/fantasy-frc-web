package discord

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdentifier_Valid(t *testing.T) {
	id := sql.NullString{String: "12345678901234567", Valid: true}
	assert.Equal(t, "<@12345678901234567>", Identifier("name", id))
}

func TestIdentifier_TooShort(t *testing.T) {
	id := sql.NullString{String: "1234567890", Valid: true}
	assert.Equal(t, "name", Identifier("name", id))
}

func TestIdentifier_Invalid(t *testing.T) {
	id := sql.NullString{String: "not-a-number", Valid: true}
	assert.Equal(t, "name", Identifier("name", id))
}

func TestIdentifier_Null(t *testing.T) {
	id := sql.NullString{Valid: false}
	assert.Equal(t, "name", Identifier("name", id))
}

func TestIsValidId(t *testing.T) {
	assert.True(t, IsValidId("12345678901234567"))
	assert.False(t, IsValidId("1234567890"))
	assert.False(t, IsValidId("not-a-number"))
	assert.False(t, IsValidId(""))
}
