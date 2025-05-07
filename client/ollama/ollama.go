package ollama

import (
	"fmt"
	"github.com/ollama/ollama/api"
)

type Ollama struct {
	Clients []*api.Client
}

func (o *Ollama) init() {
	fmt.Println("init ollama client")
	client, err := api.ClientFromEnvironment()
	if err != nil {
		panic(err)
	}
	if o.Clients == nil {
		o.Clients = make([]*api.Client, 0)
	}
	o.Clients = append(o.Clients, client)
}

func ConvertMessages(input map[string]interface{}) ([]api.Message, error) {
	// Extract the messages field
	rawMessages, ok := input["messages"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid messages format")
	}

	// Convert each message to api.Message
	var messages []api.Message
	for _, rawMessage := range rawMessages {
		msgMap, ok := rawMessage.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid message format")
		}

		// Map the fields to api.Message
		message := api.Message{
			Role:    msgMap["role"].(string),
			Content: msgMap["content"].(string),
		}
		messages = append(messages, message)
	}

	return messages, nil
}
