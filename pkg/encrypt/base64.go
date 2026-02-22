package encrypt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func EncodeBase64(input string) string {
	// Encode string to base64
	encoded := base64.StdEncoding.EncodeToString([]byte(input))
	return encoded
}

func DecodeBase64(input string) (string, error) {
	// Trim any whitespace
	input = strings.TrimSpace(input)

	// Decode base64 string
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 string: %v", err)
	}
	return string(decoded), nil
}

func DecodeBase64ToJSON(encodedStr string, target interface{}) error {
	// Trim any whitespace
	encodedStr = strings.TrimSpace(encodedStr)

	// Try standard base64 first
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedStr)
	if err != nil {
		// If standard fails, try URL-safe base64
		decodedBytes, err = base64.RawURLEncoding.DecodeString(encodedStr)
		if err != nil {
			return fmt.Errorf("base64 decode error: %v", err)
		}
	}

	// Then unmarshal JSON
	err = json.Unmarshal(decodedBytes, target)
	if err != nil {
		return fmt.Errorf("json decode error: %v", err)
	}

	return nil
}

func EncodeJSONToBase64(data interface{}) (string, error) {
	// First marshal to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("json encode error: %v", err)
	}

	// Then encode to base64
	encodedStr := base64.StdEncoding.EncodeToString(jsonBytes)
	return encodedStr, nil
}
