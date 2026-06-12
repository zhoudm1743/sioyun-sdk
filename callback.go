package sioyun

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ── 回调通知工具 ─────────────────────────────────────────────────────────

// CallbackPayload 网关回调通知的请求体。
type CallbackPayload struct {
	Event          string      `json:"event"`
	GatewayTradeNo string      `json:"gateway_trade_no"`
	OutTradeNo     string      `json:"out_trade_no"`
	EventTime      int64       `json:"event_time"`
	Data           interface{} `json:"data"`
}

// VerifyAndParseCallback 验证回调签名并解析回调数据。
// 用法：在合作伙伴的 HTTP handler 中调用，传入请求体和本地 SecretKey。
func VerifyAndParseCallback(body []byte, secretKey string, expectedEvent string) (*CallbackPayload, error) {
	var payload CallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("sioyun callback: invalid json: %w", err)
	}
	return &payload, nil
}

// VerifySignature 验签（配合 VerifyAndParseCallback 使用）。
// 签名规则：HMAC-SHA256，HMAC key = SHA256(secretKey)。
// 签名字符串 = event + "\n" + gateway_trade_no + "\n" + event_time + "\n" + SHA256(JSON(data))
func VerifySignature(secretKey string, payload *CallbackPayload, signature string) error {
	dataBytes, _ := json.Marshal(payload.Data)
	dataHash := sha256Hex(string(dataBytes))
	signStr := fmt.Sprintf("%s\n%s\n%d\n%s",
		payload.Event, payload.GatewayTradeNo, payload.EventTime, dataHash)

	// HMAC key = SHA256(secretKey)，与 HTTP 签名保持一致
	hmacKey := sha256Hex(secretKey)
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(signStr))
	expectedSign := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSign)) {
		return fmt.Errorf("sioyun callback: signature mismatch")
	}
	return nil
}

// CallbackHandler 是合作伙伴实现回调接口的参考实现。
//
// 使用方式：
//
//	http.HandleFunc("/callback/payment", sioyun.CallbackHandler(sioyun.Config{
//	    SecretKey: "sk_xxx",
//	}, func(payload *sioyun.CallbackPayload) error {
//	    fmt.Printf("收到支付回调: %+v\n", payload)
//	    // 更新本地订单状态...
//	    return nil
//	}))
func CallbackHandler(cfg Config, onEvent func(*CallbackPayload) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		signature := r.Header.Get("X-Gateway-Signature")
		if signature == "" {
			http.Error(w, "missing signature", http.StatusBadRequest)
			return
		}

		payload, err := VerifyAndParseCallback(body, cfg.CallbackSecret, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := VerifySignature(cfg.CallbackSecret, payload, signature); err != nil {
			http.Error(w, "signature verification failed", http.StatusForbidden)
			return
		}

		if err := onEvent(payload); err != nil {
			http.Error(w, "handle event failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}
}
