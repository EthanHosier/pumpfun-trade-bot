package openai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type JsonDataType int

const (
	JSONObj JsonDataType = iota
	JSONArray
)

// ExtractJsonData extracts a JSON object or array from a string
func ExtractJsonData(jsonString string, typ JsonDataType) (string, error) {
	// Remove new lines and excessive spaces
	jsonString = cleanJsonString(jsonString)

	open, close := getBrackets(typ)

	// Find the start and end of the JSON data
	start := strings.Index(jsonString, open)
	if start == -1 {
		return "", fmt.Errorf("no opening bracket found")
	}

	end := strings.LastIndex(jsonString, close)
	if end == -1 || end <= start {
		return "", fmt.Errorf("no closing bracket found")
	}

	// Extract the potential JSON data
	result := jsonString[start : end+1]

	// Validate JSON structure
	if isValidJson(result, typ) {
		return result, nil
	}
	return "", fmt.Errorf("invalid JSON data")
}

func cleanJsonString(input string) string {
	// Define regex patterns for matching the cases
	pattern := `(\[\s*")|(")\s*(\])|(\{\s*")|(")\s*(\})`

	// Replace newlines, carriage returns, and tabs with nothing in matched patterns
	re := regexp.MustCompile(pattern)
	cleanedString := re.ReplaceAllStringFunc(input, func(match string) string {
		// Replace newlines, carriage returns, and tabs within the matched pattern
		match = strings.ReplaceAll(match, "\n", "")
		match = strings.ReplaceAll(match, "\r", "")
		match = strings.ReplaceAll(match, "\t", "")
		return match
	})

	// Replace multiple spaces with a single space
	cleanedString = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedString, " ")

	return cleanedString
}

func getBrackets(typ JsonDataType) (string, string) {
	if typ == JSONArray {
		return "[", "]"
	}
	return "{", "}"
}

// isValidJson checks if the data is valid JSON and of the expected type
func isValidJson(data string, typ JsonDataType) bool {
	var js json.RawMessage
	if err := json.Unmarshal([]byte(data), &js); err != nil {
		return false
	}

	// Check if the JSON matches the expected type
	if typ == JSONObj {
		return isJSONObject(data)
	} else if typ == JSONArray {
		return isJSONArray(data)
	}
	return false
}

func isJSONObject(data string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(data), &js) == nil
}

func isJSONArray(data string) bool {
	var js []interface{}
	return json.Unmarshal([]byte(data), &js) == nil
}
