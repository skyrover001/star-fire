package public

const KEEPALIVE = "keepalive"
const REGISTER = "register"
const MESSAGE = "message"
const INCOME = "income"
const RECONNECT = "reconnect"
const MESSAGE_STREAM = "stream"
const CLOSE = "close"
const MODEL_ERROR = "model_error"
const EMBEDDING_RESPONSE = "embedding_response"
const EMBEDDING_REQUEST = "embedding_request"
const MODEL_PRICE_UPDATE = "model_price_update"

const PING = "ping"
const PONG = "pong"
const MAXLATENCE = 30000
const KEEPALIVE_TIME = 5
const CHAT_MAX_TIME = 180

const MAX_CHAT_RETRY = 3            // 最大重试次数
const CHAT_RETRY_BASE_DELAY = 100   // 重试基础延迟(ms)，指数退避
const CHAT_RETRY_TOTAL_TIMEOUT = 10 // 重试总超时(秒)

const ABORT = "abort" // 取消消息标记：server 放弃某请求时通知 client 停止处理

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

type ModelPriceUpdate struct {
	Model string  `json:"model"`
	IPPM  float64 `json:"ippm"`
	OPPM  float64 `json:"oppm"`
	CIPPM float64 `json:"cippm"`
}

func ISStrINArray(str string, arr []string) bool {
	for _, s := range arr {
		if str == s {
			return true
		}
	}
	return false
}
