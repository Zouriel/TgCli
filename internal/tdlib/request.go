package tdlib

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type tdlibError struct {
	Type    string `json:"@type"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Type  string          `json:"@type"`
	Extra json.RawMessage `json:"@extra,omitempty"`
}

var receiveMu sync.Mutex

func SendRequestAndWait(tdjson *TDJSON, clientID int32, requestJSON string, extraTag string, timeout time.Duration) (string, error) {
	var requestMap map[string]any
	if err := json.Unmarshal([]byte(requestJSON), &requestMap); err != nil {
		return "", err
	}
	requestMap["@extra"] = extraTag

	requestBytes, err := json.Marshal(requestMap)
	if err != nil {
		return "", err
	}

	if err := tdjson.Send(clientID, string(requestBytes)); err != nil {
		return "", err
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		receiveMu.Lock()
		updateJSON, err := tdjson.Receive(1.0)
		receiveMu.Unlock()

		if err != nil {
			return "", err
		}
		if updateJSON == "" {
			continue
		}

		var env envelope
		if err := json.Unmarshal([]byte(updateJSON), &env); err != nil {
			continue
		}
		if len(env.Extra) == 0 {
			continue
		}

		var extraValue string
		if err := json.Unmarshal(env.Extra, &extraValue); err != nil {
			continue
		}
		if extraValue != extraTag {
			continue
		}

		if env.Type == "error" {
			var tdErr tdlibError
			if err := json.Unmarshal([]byte(updateJSON), &tdErr); err == nil {
				return "", fmt.Errorf("tdlib error %d: %s", tdErr.Code, tdErr.Message)
			}
			return "", fmt.Errorf("tdlib error: %s", updateJSON)
		}

		return updateJSON, nil
	}

	return "", errors.New("timeout waiting for TDLib response")
}
