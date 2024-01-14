package helpers

import (
	"encoding/json"
	"io"
)

const (
	UNIDENTIFIED_ERROR = "Unidentified error, please contact support."
)

type ApiError struct {
	Error   bool        `json:"error"`
	Message interface{} `json:"message"`
}

func ErrorFromAPIBody(body io.ReadCloser) string {
	r, err := io.ReadAll(body)

	if err != nil {
		return UNIDENTIFIED_ERROR
	}

	var error ApiError

	if err := json.Unmarshal(r, &error); err != nil {
		return UNIDENTIFIED_ERROR
	}

	switch msg := error.Message.(type) {
	case string:
		return msg
	case map[string]interface{}:
		if msgVal, ok := msg["msg"].(string); ok {
			return msgVal
		}
	}

	return UNIDENTIFIED_ERROR
}
