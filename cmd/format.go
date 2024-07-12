package cmd

import (
	"bytes"
	"encoding/json"
)

func ToJsonString(data interface{}) (string, error) {
	var prettyJSON bytes.Buffer
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	prettyJSON.Write(jsonBytes)
	return prettyJSON.String(), nil
}
