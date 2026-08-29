package public

// 上游/用户 API 格式常量。server 与 client 共享：
//   - client 上报模型时把 Backend.Format 填入 Model.UpstreamFormat；
//   - server 解析用户请求、读取 client.UpstreamFormat 时使用。
const (
	FormatOpenAI    = "openai"    // OpenAI Chat Completions
	FormatAnthropic = "anthropic" // Anthropic Messages
	FormatResponses = "responses" // OpenAI Responses
)

// validFormats 是所有受支持的格式列表。
var validFormats = []string{FormatOpenAI, FormatAnthropic, FormatResponses}

// ValidFormats 返回所有受支持的格式。
func ValidFormats() []string {
	return validFormats
}

// IsValidFormat 判断格式是否受支持。
func IsValidFormat(format string) bool {
	return ISStrINArray(format, validFormats)
}
