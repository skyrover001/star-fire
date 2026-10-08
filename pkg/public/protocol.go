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

const LATENCY_EXCEEDED = "latency_exceeded" // 通知 client：网络延迟过高，暂不采纳其模型算力

// smart 负载均衡算法目标参数
const LB_TARGET_ONLINE_SEC = 3600        // 目标在线时长（1小时），用于在线稳定性评分
const LB_TARGET_TOKENS_PER_HOUR = 100000 // 目标产能（token/小时），用于服务等级评分

type WSMessage struct {
	Type        string      `json:"type"`
	Content     interface{} `json:"content"`
	FingerPrint string      `json:"fingerprint"`
	// Format 是本次消息的协议格式（openai | anthropic | responses）。
	// server 把 Canonical 转成 client 上游格式后填入，client 据此选择引擎。
	// omitempty：老 client 序列化的消息不带此字段，缺省视为 openai。
	Format string `json:"format,omitempty"`
}

type PPMessage struct {
	Type            string   `json:"type"`
	Timestamp       string   `json:"timestamp"`
	AvailableModels []*Model `json:"update_model"`
	// 客户端上报的上行带宽（Mbps），用于 smart 负载均衡带宽维度。
	// 客户端在启动/心跳时上报，server 端据此评估该 client 到 server 的上行带宽。
	BandwidthMbps float64 `json:"bandwidth_mbps,omitempty"`
	// 客户端自定义连接数上限（0 = 使用会员等级默认上限）。
	// 由 Python 客户端 app 通过滑块配置（0 ~ 会员上限），经 Go 客户端上报到 server。
	MaxConnections int `json:"max_connections,omitempty"`
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
