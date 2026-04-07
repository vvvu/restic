package backend

import (
	"unicode"

	"github.com/restic/restic/internal/errors"
)

// SplitShellStrings returns the list of shell strings from a shell command string.
// It supports single and double quoted strings and backslash escaping.
// Backslash escapes the following character (making it literal), and the backslash
// itself is removed from the output.
func SplitShellStrings(data string) (strs []string, err error) {
	var quote rune = 0
	var escapedNext bool
	var fieldStart int = -1
	var result []rune

	for i, r := range data {
		// Handle escape sequences
		if escapedNext {
			escapedNext = false
			if fieldStart == -1 {
				fieldStart = i // Start of new field
			}
			result = append(result, r)
			continue
		}

		// Backslash starts an escape sequence
		if r == '\\' {
			escapedNext = true
			if fieldStart == -1 {
				fieldStart = i // Start of new field (will include escaped char)
			}
			continue
		}

		// Handle quote ending
		if quote != 0 && r == quote {
			quote = 0
			continue
		}

		// Handle quote starting
		if quote == 0 && (r == '"' || r == '\'') {
			quote = r
			if fieldStart == -1 {
				fieldStart = i
			}
			continue
		}

		// Inside quote - not a split character
		if quote != 0 {
			if fieldStart == -1 {
				fieldStart = i
			}
			result = append(result, r)
			continue
		}

		// Outside quote - spaces split
		if unicode.IsSpace(r) {
			if fieldStart >= 0 {
				strs = append(strs, string(result))
				result = nil
				fieldStart = -1
			}
			continue
		}

		// Regular character
		if fieldStart == -1 {
			fieldStart = i
		}
		result = append(result, r)
	}

	// Handle remaining content
	if fieldStart >= 0 {
		strs = append(strs, string(result))
	}

	// Check for errors
	switch {
	case quote == '\'':
		return nil, errors.New("single-quoted string not terminated")
	case quote == '"':
		return nil, errors.New("double-quoted string not terminated")
	case escapedNext:
		return nil, errors.New("backslash escape at end of string")
	case len(strs) == 0:
		return nil, errors.New("command string is empty")
	}

	return strs, nil
}
