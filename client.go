package sioyun

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ── 配置 ─────────────────────────────────────────────────────────────────

// Config SDK 配置。
type Config struct {
	// BaseURL 网关根地址，如 https://www.sioyun.com/api/gateway/v1
	BaseURL string
	// AccessKey 访问标识符（ak_ 前缀）
	AccessKey string
	// SecretKey 签名密钥（sk_ 前缀）
	SecretKey string
	// Timeout HTTP 超时（秒），默认 30
	Timeout int
	// CallbackSecret 回调验签所用的 SecretKey（用于 HandleCallback）
	// 默认等于 SecretKey，如果业务方有独立的回调密钥可单独设置
	CallbackSecret string
	// Transport 自定义 HTTP Transport（nil 则用默认）
	Transport http.RoundTripper
}

// ── 客户端 ───────────────────────────────────────────────────────────────

// Client 网关客户端。
// 通过 New() 创建，内部维护连接池，线程安全。
type Client struct {
	cfg        Config
	http       *http.Client
	base       string
	pathPrefix string // BaseURL 的路径部分（如 /api/gateway/v1），用于签名时拼接完整路径
	sms        *SmsService
	pay        *PaymentService
	pt         *PartnerService
	app        *AppService
}

// New 创建网关客户端，初始化时发送测试请求验证连通性。
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("sioyun: BaseURL is required")
	}
	if cfg.AccessKey == "" {
		return nil, fmt.Errorf("sioyun: AccessKey is required")
	}
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("sioyun: SecretKey is required")
	}
	if !strings.HasPrefix(cfg.BaseURL, "http") {
		cfg.BaseURL = "https://" + cfg.BaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}
	if cfg.CallbackSecret == "" {
		cfg.CallbackSecret = cfg.SecretKey
	}

	c := &Client{
		cfg:  cfg,
		base: cfg.BaseURL,
		http: &http.Client{
			Timeout:   time.Duration(cfg.Timeout) * time.Second,
			Transport: cfg.Transport,
		},
	}
	// 从 BaseURL 提取路径前缀，用于签名时拼接完整请求路径
	if u, err := url.Parse(cfg.BaseURL); err == nil {
		c.pathPrefix = u.Path
	}
	c.sms = &SmsService{client: c}
	c.pay = &PaymentService{client: c}
	c.pt = &PartnerService{client: c}
	c.app = &AppService{client: c}

	// 连通性验证
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var result APIResponse
	if err := c.do(ctx, "GET", "/sms/balance", nil, &result); err != nil {
		return nil, fmt.Errorf("sioyun: connectivity check failed: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("sioyun: connectivity check returned code=%d msg=%s", result.Code, result.Msg)
	}

	return c, nil
}

// SMS 返回短信服务。
func (c *Client) SMS() *SmsService { return c.sms }

// Pay 返回支付服务。
func (c *Client) Pay() *PaymentService { return c.pay }

// Partner 返回进件服务。
func (c *Client) Partner() *PartnerService { return c.pt }

// App 返回应用服务。
func (c *Client) App() *AppService { return c.app }

// Config 返回当前配置（只读副本）。
func (c *Client) Config() Config { return c.cfg }

// ── 内部方法 ────────────────────────────────────────────────────────────

// do 执行 HTTP 请求，自动签名和解析响应。
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyBytes []byte
	var bodyStr string
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("sioyun: marshal request body: %w", err)
		}
		bodyStr = string(bodyBytes)
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := randomString(16)
	// 签名路径使用完整路径（pathPrefix + 路由路径），与后端 sign_auth.go ctx.Path() 对齐
	signPath := c.pathPrefix + path
	signature := sign(c.cfg.SecretKey, method, signPath, timestamp, nonce, bodyStr)

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("sioyun: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Access-Key", c.cfg.AccessKey)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Request-Id", newID())

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sioyun: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("sioyun: read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return fmt.Errorf("sioyun: unmarshal response: %w (body=%s)", err, string(respBytes))
	}

	if resp.StatusCode != 200 || (apiResp.Code != 0 && apiResp.Code != 200) {
		return &APIError{
			HTTPStatus: resp.StatusCode,
			Code:       apiResp.Code,
			Msg:        apiResp.Msg,
		}
	}

	if result != nil && apiResp.Data != nil {
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, result); err != nil {
			return fmt.Errorf("sioyun: unmarshal data: %w", err)
		}
	}

	return nil
}

// ── 签名 ────────────────────────────────────────────────────────────────

// sign 计算 HMAC-SHA256 签名。
//
// 签名字符串 = METHOD + "\n" + PATH + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + BODY_SHA256
// HMAC key = SHA256(secretKey)
// 签名 = Hex(HMAC-SHA256(signStr, HMAC_key))
//
// 注意：与后台 sign_auth.go 保持一致，客户端用 SHA256(secretKey) 作为 HMAC key。
func sign(secretKey, method, path, timestamp, nonce, body string) string {
	bodyHash := sha256Hex(body)
	signStr := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", method, path, timestamp, nonce, bodyHash)

	// HMAC key = SHA256(secretKey)，与后台 sign_auth.go 第 30-31 行注释一致
	hmacKey := sha256Hex(secretKey)
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(signStr))
	return hex.EncodeToString(mac.Sum(nil))
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func randomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}
