package public

const KEEPALIVE = "keepalive"
const REGISTER = "register"
const MESSAGE = "message"
const RECONNECT = "reconnect"
const MESSAGE_STREAM = "stream"
const CLOSE = "close"
const MODEL_ERROR = "model_error"

const PING = "ping"
const PONG = "pong"
const MAXLATENCE = 65535
const KEEPALIVE_TIME = 5
const CHAT_MAX_TIME = 180

type WSMessage struct {
	Type        string      `json:"type"`
	Content     interface{} `json:"content"`
	FingerPrint string      `json:"fingerprint"`
}

type PPMessage struct {
	Type            string   `json:"type"`
	Timestamp       string   `json:"timestamp"`
	AvailableModels []*Model `json:"update_model"`
}

func ISStrINArray(str string, arr []string) bool {
	for _, s := range arr {
		if str == s {
			return true
		}
	}
	return false
}
