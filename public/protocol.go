package public

const KEEPALIVE = "keepalive"
const REGISTER = "register"
const MESSAGE = "message"
const CLOSE = "close"

const PING = "ping"
const PONG = "pong"
const MAXLATENCE = 65535

type WSMessage struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

type PPMessage struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}
