package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type StringArray []string

func (s *StringArray) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*s = parsePGTextArray(string(v))
	case string:
		*s = parsePGTextArray(v)
	default:
		return fmt.Errorf("unsupported type for StringArray: %T", value)
	}
	return nil
}

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	escaped := make([]string, len(s))
	for i, v := range s {
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `"`, `\"`)
		escaped[i] = `"` + v + `"`
	}
	return "{" + strings.Join(escaped, ",") + "}", nil
}

func parsePGTextArray(input string) []string {
	input = strings.TrimSpace(input)
	if len(input) < 2 || input[0] != '{' || input[len(input)-1] != '}' {
		return nil
	}
	input = input[1 : len(input)-1]

	var result []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes && ch == ',' {
			result = append(result, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(ch)
	}
	result = append(result, current.String())

	clean := make([]string, 0, len(result))
	for _, s := range result {
		s = strings.TrimSpace(s)
		if s != "" {
			clean = append(clean, s)
		}
	}
	return clean
}
