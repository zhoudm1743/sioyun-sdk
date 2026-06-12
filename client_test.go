package sioyun

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSign(t *testing.T) {
	sig := sign("sk_test123", "POST", "/sms/send", "1718150400", "abc123def456", `{"phone":"13800138000"}`)
	if sig == "" {
		t.Fatal("signature is empty")
	}
	t.Logf("signature: %s", sig)
}

func TestClientSmsSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		if r.Header.Get("X-Access-Key") == "" {
			t.Error("missing X-Access-Key")
		}
		if r.Header.Get("X-Signature") == "" {
			t.Error("missing X-Signature")
		}

		// 返回模拟响应
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"send_id":           "test_send_001",
				"fee":               1,
				"balance_remaining": 999,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:   server.URL,
		AccessKey: "ak_test12345678901234567890ab",
		SecretKey: "sk_test123456789012345678901234567890123456789012345678901234567890ab",
		Timeout:   10,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.SMS().Send(context.Background(), SmsSendReq{
		Phone:        "13800138000",
		TemplateCode: "verify_code",
		Params:       map[string]string{"code": "123456"},
	})
	if err != nil {
		t.Fatalf("Send() failed: %v", err)
	}
	if resp.SendID != "test_send_001" {
		t.Errorf("unexpected send_id: %s", resp.SendID)
	}
	if resp.Fee != 1 {
		t.Errorf("unexpected fee: %d", resp.Fee)
	}
}

func TestClientPayCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_trade_no":     "ORDER001",
				"gateway_trade_no": "GATEWAY001",
				"pay_method":       "wechat_jsapi",
				"amount":           100,
				"pay_info": map[string]string{
					"appId":     "wx_test",
					"timeStamp": "1718150400",
					"package":   "prepay_id=wx_test_001",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:   server.URL,
		AccessKey: "ak_test",
		SecretKey: "sk_test",
		Timeout:   10,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().Create(context.Background(), OrderCreateReq{
		OutTradeNo:  "ORDER001",
		Amount:      100,
		PayMethod:   "wechat_jsapi",
		Description: "test",
		NotifyURL:   "https://example.com/cb",
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	if resp.GatewayTradeNo != "GATEWAY001" {
		t.Errorf("unexpected gateway_trade_no: %s", resp.GatewayTradeNo)
	}
}

func TestIsInsufficientFunds(t *testing.T) {
	err := &APIError{Code: 402, Msg: "短信额度不足"}
	if !IsInsufficientFunds(err) {
		t.Error("IsInsufficientFunds should return true for code 402")
	}
}

func TestIsRateLimited(t *testing.T) {
	err := &APIError{Code: 429, Msg: "频率超限"}
	if !IsRateLimited(err) {
		t.Error("IsRateLimited should return true for code 429")
	}
}

func TestSignConsistency(t *testing.T) {
	// 同一个输入产生相同的签名（确定性）
	sig1 := sign("sk_test", "POST", "/sms/send", "1000000", "nonce1", `{"phone":"13800138000"}`)
	sig2 := sign("sk_test", "POST", "/sms/send", "1000000", "nonce1", `{"phone":"13800138000"}`)
	if sig1 != sig2 {
		t.Errorf("signatures should be deterministic: %s != %s", sig1, sig2)
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{HTTPStatus: 400, Code: 400, Msg: "参数错误"}
	expected := "sioyun: [400] 参数错误 (http=400)"
	if err.Error() != expected {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}
