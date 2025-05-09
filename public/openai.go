package public

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model            string          `json:"model"`
	Messages         []OpenAIMessage `json:"messages"`
	Temperature      float64         `json:"temperature"`
	TopP             float64         `json:"top_p"`
	N                int             `json:"n"`
	Stream           bool            `json:"stream"`
	Stop             []string        `json:"stop"`
	MaxTokens        int             `json:"max_tokens"`
	PresencePenalty  float64         `json:"presence_penalty"`
	FrequencyPenalty float64         `json:"frequency_penalty"`
	User             string          `json:"user"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	FinishReason string        `json:"finish_reason"`
	Message      OpenAIMessage `json:"message"`
}
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int            `json:"created"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type Model struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size string `json:"size"`
	Arch string `json:"arch"`
}
