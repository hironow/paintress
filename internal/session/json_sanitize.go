package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// SanitizeJSONFile reads a file that should contain JSON, strips any
// non-JSON text wrapping (markdown code blocks, natural language
// prefixes/suffixes) that Claude may add, and returns clean JSON bytes.
func SanitizeJSONFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sanitize read: %w", err)
	}
	cleaned := stripMarkdownCodeBlock(data)
	var v any
	if err := json.Unmarshal(cleaned, &v); err != nil {
		extracted := extractJSON(cleaned)
		if err2 := json.Unmarshal(extracted, &v); err2 != nil {
			return nil, fmt.Errorf("sanitize unmarshal: %w", err)
		}
		cleaned = extracted
	}
	return cleaned, nil
}

// stripMarkdownCodeBlock removes markdown code block wrappers (```json ... ```)
// from data. Claude may wrap JSON output in markdown fences.
func stripMarkdownCodeBlock(data []byte) []byte {
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte("```")) {
		return trimmed
	}
	if idx := bytes.IndexByte(trimmed, '\n'); idx >= 0 {
		trimmed = trimmed[idx+1:]
	} else {
		return trimmed
	}
	if idx := bytes.LastIndex(trimmed, []byte("```")); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return bytes.TrimSpace(trimmed)
}

// extractJSON finds the first top-level JSON value ({...} or [...]) in data
// by scanning for the opening delimiter and matching it with the closing one.
// Handles cases where Claude wraps JSON in natural language text.
func extractJSON(data []byte) []byte {
	// Find first { or [
	startObj := bytes.IndexByte(data, '{')
	startArr := bytes.IndexByte(data, '[')
	start := -1
	var open, close byte
	switch {
	case startObj >= 0 && (startArr < 0 || startObj < startArr):
		start, open, close = startObj, '{', '}'
	case startArr >= 0:
		start, open, close = startArr, '[', ']'
	default:
		return data
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(data); i++ {
		if escaped {
			escaped = false
			continue
		}
		c := data[i]
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return data[start : i+1]
			}
		}
	}
	return data[start:]
}
