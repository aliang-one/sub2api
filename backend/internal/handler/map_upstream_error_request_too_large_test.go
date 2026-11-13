package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 上游对超大请求体（如客户端把 2MB 级图片原样 base64 内联后 5~6MB 的请求）
// 返回 413 时，网关必须回传 413 + 明确的 request_too_large 语义，而不是落进
// mapUpstreamError 的 default 分支伪装成 502 "Upstream request failed"。
// 背景：2026-11-12~13 视频会话连续 7 次 502，客户端排查半天才发现真实原因
// 是请求体过大——错误码不透明让这个问题的定位成本高了一个数量级。
//
// 参考 Anthropic 官方 API：超限请求回 413 + type=request_too_large，这里对齐。
func TestMapUpstreamErrorRequestTooLarge(t *testing.T) {
	h := &GatewayHandler{}

	status, errType, msg := h.mapUpstreamError(http.StatusRequestEntityTooLarge)

	assert.Equal(t, http.StatusRequestEntityTooLarge, status)
	assert.Equal(t, "request_too_large", errType)
	assert.Contains(t, msg, "Request body too large")
	assert.Contains(t, msg, "reduce")
}

// OpenAI 兼容路径同样要 413 透传（同一网关的另一个入口，客户端共享同一类故障）。
func TestOpenAIMapUpstreamErrorRequestTooLarge(t *testing.T) {
	h := &OpenAIGatewayHandler{}

	status, errType, msg := h.mapUpstreamError(http.StatusRequestEntityTooLarge)

	assert.Equal(t, http.StatusRequestEntityTooLarge, status)
	assert.Equal(t, "request_too_large", errType)
	assert.Contains(t, msg, "Request body too large")
}

// Gemini 路径：413 状态码透传 + 明确文案（该处理器没有 errType 概念）。
func TestMapGeminiUpstreamErrorRequestTooLarge(t *testing.T) {
	status, msg := mapGeminiUpstreamError(http.StatusRequestEntityTooLarge)

	assert.Equal(t, http.StatusRequestEntityTooLarge, status)
	assert.Contains(t, msg, "Request body too large")
}

// 防回归：default 分支的既有行为不变（未列举状态码仍走 502 兜底）。
func TestMapUpstreamErrorDefaultUnchanged(t *testing.T) {
	h := &GatewayHandler{}

	status, errType, msg := h.mapUpstreamError(http.StatusTeapot)

	assert.Equal(t, http.StatusBadGateway, status)
	assert.Equal(t, "upstream_error", errType)
	assert.Equal(t, "Upstream request failed", msg)
}
