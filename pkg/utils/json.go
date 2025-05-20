package utils

import (
	"encoding/json"
	"fmt"
)

func ParseJSON(input interface{}, output interface{}) error {
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}
	err = json.Unmarshal(data, output)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	return nil
}
