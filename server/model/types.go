package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// StringArray is a Postgres text[] value that can be scanned from and written
// to the database using only the standard library.
type StringArray []string

// Scan implements sql.Scanner for text[].
func (a *StringArray) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}

	switch v := value.(type) {
	case string:
		parsed, err := parsePostgresTextArray(v)
		if err != nil {
			return err
		}
		*a = parsed
		return nil
	case []byte:
		parsed, err := parsePostgresTextArray(string(v))
		if err != nil {
			return err
		}
		*a = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into StringArray", value)
	}
}

// Value implements driver.Valuer for text[].
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	parts := make([]string, len(a))
	for i, s := range a {
		parts[i] = `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

func parsePostgresTextArray(s string) ([]string, error) {
	if s == "{}" {
		return []string{}, nil
	}
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return nil, fmt.Errorf("invalid text array format")
	}

	inner := s[1 : len(s)-1]
	var result []string
	var current strings.Builder
	inQuote := false
	escaped := false

	for _, r := range inner {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inQuote = !inQuote
			continue
		}
		if r == ',' && !inQuote {
			result = append(result, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result, nil
}
