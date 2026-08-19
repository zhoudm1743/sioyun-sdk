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

func TestSignPathWithGatewayPrefix(t *testing.T) {
	const (
		secretKey  = "sk_test123456789012345678901234567890123456789012345678901234567890ab"
		accessKey  = "ak_test12345678901234567890ab"
		pathPrefix = "/api/gateway/v1"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pathPrefix+"/sms/balance" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		ts := r.Header.Get("X-Timestamp")
		nonce := r.Header.Get("X-Nonce")
		gotSig := r.Header.Get("X-Signature")
		wantSig := sign(secretKey, r.Method, r.URL.Path, ts, nonce, "")

		if gotSig != wantSig {
			t.Errorf("signature mismatch: got=%s want=%s (path=%s)", gotSig, wantSig, r.URL.Path)
		}

		resp := APIResponse{
			Code: 200,
			Msg:  "ok",
			Data: map[string]interface{}{
				"total_remaining": int64(100),
				"packages":        []any{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:   server.URL + pathPrefix,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Timeout:   10,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	balance, err := client.SMS().Balance(context.Background())
	if err != nil {
		t.Fatalf("Balance() failed: %v", err)
	}
	if balance.TotalRemaining != 100 {
		t.Errorf("unexpected total_remaining: %d", balance.TotalRemaining)
	}
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
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
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
		OpenID:      "oTestOpenId",
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	if resp.GatewayTradeNo != "GATEWAY001" {
		t.Errorf("unexpected gateway_trade_no: %s", resp.GatewayTradeNo)
	}
	jsapi, err := resp.WechatJsapiPayInfo()
	if err != nil {
		t.Fatalf("WechatJsapiPayInfo() failed: %v", err)
	}
	if jsapi.Package != "prepay_id=wx_test_001" {
		t.Errorf("unexpected package: %s", jsapi.Package)
	}
}

func TestClientPayCreateMethods(t *testing.T) {
	cases := []struct {
		method  string
		payInfo map[string]interface{}
		check   func(t *testing.T, resp *OrderCreateResp)
	}{
		{
			method: "wechat_jsapi",
			payInfo: map[string]interface{}{
				"appId": "wx_test", "timeStamp": "1718150400",
				"package": "prepay_id=wx_test_001", "paySign": "sig",
			},
			check: func(t *testing.T, resp *OrderCreateResp) {
				info, err := resp.WechatJsapiPayInfo()
				if err != nil || info.Package == "" {
					t.Fatalf("WechatJsapiPayInfo() failed: %v", err)
				}
			},
		},
		{
			method:  "wechat_h5",
			payInfo: map[string]interface{}{"h5_url": "https://wx.test/h5"},
			check: func(t *testing.T, resp *OrderCreateResp) {
				url, err := resp.WechatH5PayInfo()
				if err != nil || url == "" {
					t.Fatalf("WechatH5PayInfo() failed: %v", err)
				}
			},
		},
		{
			method:  "wechat_native",
			payInfo: map[string]interface{}{"code_url": "weixin://wxpay/bizpayurl?pr=abc"},
			check: func(t *testing.T, resp *OrderCreateResp) {
				url, err := resp.WechatNativePayInfo()
				if err != nil || url == "" {
					t.Fatalf("WechatNativePayInfo() failed: %v", err)
				}
			},
		},
		{
			method: "wechat_app",
			payInfo: map[string]interface{}{
				"appId": "wx_test", "partnerId": "123", "prepayId": "wx_prepay",
				"packageValue": "Sign=WXPay", "nonceStr": "n", "timeStamp": "1", "sign": "s",
			},
			check: func(t *testing.T, resp *OrderCreateResp) {
				info, err := resp.WechatAppPayInfo()
				if err != nil || info.PrepayID == "" {
					t.Fatalf("WechatAppPayInfo() failed: %v", err)
				}
			},
		},
		{
			method:  "alipay_qr",
			payInfo: map[string]interface{}{"qr_code": "https://qr.alipay.com/abc"},
			check: func(t *testing.T, resp *OrderCreateResp) {
				code, err := resp.AlipayQrPayInfo()
				if err != nil || code == "" {
					t.Fatalf("AlipayQrPayInfo() failed: %v", err)
				}
			},
		},
		{
			method:  "alipay_h5",
			payInfo: map[string]interface{}{"h5_url": "https://openapi.alipay.com/gateway.do?..."},
			check: func(t *testing.T, resp *OrderCreateResp) {
				url, err := resp.AlipayH5PayInfo()
				if err != nil || url == "" {
					t.Fatalf("AlipayH5PayInfo() failed: %v", err)
				}
			},
		},
		{
			method:  "alipay_app",
			payInfo: map[string]interface{}{"order_string": "app_id=2021&method=alipay.trade.app.pay"},
			check: func(t *testing.T, resp *OrderCreateResp) {
				s, err := resp.AlipayAppPayInfo()
				if err != nil || s == "" {
					t.Fatalf("AlipayAppPayInfo() failed: %v", err)
				}
			},
		},
		{
			method:  "wechat_micropay",
			payInfo: map[string]interface{}{"trade_state": "SUCCESS", "transaction_id": "420000001", "pay_amount": 100, "pay_time": int64(1718150500)},
			check: func(t *testing.T, resp *OrderCreateResp) {
				info, err := resp.MicroPayInfo()
				if err != nil || info.TradeState != "SUCCESS" {
					t.Fatalf("MicroPayInfo() failed: %v", err)
				}
			},
		},
		{
			method:  "alipay_micropay",
			payInfo: map[string]interface{}{"trade_state": "SUCCESS", "trade_no": "2026061200001", "pay_amount": 100, "pay_time": int64(1718150500)},
			check: func(t *testing.T, resp *OrderCreateResp) {
				info, err := resp.MicroPayInfo()
				if err != nil || info.TradeNo == "" {
					t.Fatalf("MicroPayInfo() failed: %v", err)
				}
			},
		},
		{
			method:  "unionpay_micropay",
			payInfo: map[string]interface{}{"trade_state": "SUCCESS", "target_order_id": "2026061200002", "pay_amount": 100, "pay_time": int64(1718150500)},
			check: func(t *testing.T, resp *OrderCreateResp) {
				info, err := resp.MicroPayInfo()
				if err != nil || info.TradeState != "SUCCESS" {
					t.Fatalf("MicroPayInfo() failed: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := APIResponse{
					Code: 0,
					Msg:  "success",
					Data: map[string]interface{}{
						"out_trade_no":     "ORDER001",
						"gateway_trade_no": "GATEWAY001",
						"pay_method":       tc.method,
						"amount":           100,
						"pay_info":         tc.payInfo,
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client, err := New(Config{
				BaseURL:               server.URL,
				AccessKey:             "ak_test",
				SecretKey:             "sk_test",
				Timeout:               10,
				SkipConnectivityCheck: true,
			})
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			resp, err := client.Pay().Create(context.Background(), OrderCreateReq{
				OutTradeNo:  "ORDER001",
				Amount:      100,
				PayMethod:   tc.method,
				Description: "test",
				NotifyURL:   "https://example.com/cb",
			})
			if err != nil {
				t.Fatalf("Create() failed: %v", err)
			}
			if resp.PayMethod != tc.method {
				t.Errorf("unexpected pay_method: %s", resp.PayMethod)
			}
			tc.check(t, resp)
		})
	}
}

func TestClientPayQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/pay/query/ORDER001" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_trade_no":     "ORDER001",
				"gateway_trade_no": "GATEWAY001",
				"status":           "SUCCESS",
				"pay_method":       "wechat_jsapi",
				"amount":           100,
				"pay_amount":       100,
				"pay_time":         int64(1718150500),
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().Query(context.Background(), OrderQueryReq{
		OutTradeNo: "ORDER001",
	})
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}
	if resp.Status != "SUCCESS" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func TestClientPayClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/pay/close" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_trade_no": "ORDER001",
				"status":       "CLOSED",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().Close(context.Background(), OrderCloseReq{OutTradeNo: "ORDER001"})
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	if resp.Status != "CLOSED" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func TestClientPayRefund(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/pay/refund" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_refund_no": "REFUND001",
				"refund_id":     "503000123456789",
				"refund_amount": 50,
				"status":        "PROCESSING",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().Refund(context.Background(), RefundCreateReq{
		OutTradeNo:   "ORDER001",
		OutRefundNo:  "REFUND001",
		RefundAmount: 50,
	})
	if err != nil {
		t.Fatalf("Refund() failed: %v", err)
	}
	if resp.RefundID != "503000123456789" {
		t.Errorf("unexpected refund_id: %s", resp.RefundID)
	}
}

func TestClientPayRefundQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/pay/refund/query" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_refund_no": "REFUND001",
				"refund_id":     "503000123456789",
				"out_trade_no":  "ORDER001",
				"refund_amount": 50,
				"status":        "SUCCESS",
				"refund_time":   int64(1718151000),
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().RefundQuery(context.Background(), RefundQueryReq{OutRefundNo: "REFUND001"})
	if err != nil {
		t.Fatalf("RefundQuery() failed: %v", err)
	}
	if resp.Status != "SUCCESS" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func TestClientPaySplit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/pay/split" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_profit_share_no": "SPLIT001",
				"channel":             "wechat",
				"amount":              int64(100),
				"status":              "PROCESSING",
				"receivers": []map[string]interface{}{
					{
						"receiver_type": "MERCHANT_ID",
						"account":       "1900000109",
						"amount":        int64(3),
						"description":   "",
						"result":        "",
						"fail_reason":   "",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().Split(context.Background(), SplitCreateReq{
		OutTradeNo:       "ORDER001",
		OutProfitShareNo: "SPLIT001",
		Amount:           100,
	})
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}
	if resp.Status != "PROCESSING" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
	if len(resp.Receivers) != 1 || resp.Receivers[0].Account != "1900000109" {
		t.Errorf("unexpected receivers: %+v", resp.Receivers)
	}
}

func TestClientPaySplitQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/pay/split/query/SPLIT001" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_profit_share_no": "SPLIT001",
				"out_trade_no":        "ORDER001",
				"channel":             "wechat",
				"amount":              int64(100),
				"status":              "SUCCESS",
				"channel_record_no":   "3008450740201411110007820472",
				"receivers":           []map[string]interface{}{},
				"profit_share_time":   int64(1718151000),
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().SplitQuery(context.Background(), "SPLIT001")
	if err != nil {
		t.Fatalf("SplitQuery() failed: %v", err)
	}
	if resp.Status != "SUCCESS" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
	if resp.ChannelRecordNo != "3008450740201411110007820472" {
		t.Errorf("unexpected channel_record_no: %s", resp.ChannelRecordNo)
	}
}

func TestClientPaySplitReturn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/pay/split/return" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_return_no": "RETURN001",
				"return_no":     "3008450740201411110007820473",
				"return_amount": int64(3),
				"status":        "PROCESSING",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().SplitReturn(context.Background(), SplitReturnReq{
		OutTradeNo:       "ORDER001",
		OutProfitShareNo: "SPLIT001",
		OutReturnNo:      "RETURN001",
		ReturnAmount:     3,
	})
	if err != nil {
		t.Fatalf("SplitReturn() failed: %v", err)
	}
	if resp.Status != "PROCESSING" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func TestClientPaySplitUnsplitAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/pay/split/unsplit_amount/ORDER001" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		resp := APIResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"out_trade_no":   "ORDER001",
				"unsplit_amount": int64(97),
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL:               server.URL,
		AccessKey:             "ak_test",
		SecretKey:             "sk_test",
		Timeout:               10,
		SkipConnectivityCheck: true,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	resp, err := client.Pay().SplitUnsplitAmount(context.Background(), "ORDER001")
	if err != nil {
		t.Fatalf("SplitUnsplitAmount() failed: %v", err)
	}
	if resp.UnsplitAmount != 97 {
		t.Errorf("unexpected unsplit_amount: %d", resp.UnsplitAmount)
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
