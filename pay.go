package sioyun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// PaymentService 支付服务。
type PaymentService struct {
	client *Client
}

// ── 下单 ─────────────────────────────────────────────────────────────────

// OrderCreateReq 支付下单请求。
type OrderCreateReq struct {
	OutTradeNo    string `json:"out_trade_no"`              // 必填：商户订单号（唯一）
	Amount        int64  `json:"amount"`                    // 必填：金额（分）
	PayMethod     string `json:"pay_method"`                // 必填：wechat_jsapi / wechat_h5 / wechat_native / wechat_app / alipay_qr / alipay_h5 / alipay_app / unionpay_qr / unionpay_mini / unionpay_jsapi
	Description   string `json:"description"`               // 必填：商品描述
	NotifyURL     string `json:"notify_url"`                // 必填：支付结果回调地址
	OpenID        string `json:"openid,omitempty"`          // 条件：微信 jsapi 支付必填
	SubMchID      string `json:"sub_mchid,omitempty"`       // 指定子商户号
	Attach        string `json:"attach,omitempty"`          // 附加数据（回调原样返回）
	ExpireMinutes int    `json:"expire_minutes,omitempty"`  // 过期分钟数
	ClientIP      string `json:"client_ip,omitempty"`       // 条件：wechat_h5 建议传入
	ReturnURL     string `json:"return_url,omitempty"`      // 可选：alipay_h5 支付完成跳转
}

// OrderCreateResp 支付下单响应。
type OrderCreateResp struct {
	OutTradeNo     string                 `json:"out_trade_no"`
	GatewayTradeNo string                 `json:"gateway_trade_no"`
	PayMethod      string                 `json:"pay_method"`
	Amount         int64                  `json:"amount"`
	PayInfo        map[string]interface{} `json:"pay_info"`
}

// WechatJsapiPayInfo 微信 JSAPI 调起支付参数（用于 wx.requestPayment / WeixinJSBridge）。
type WechatJsapiPayInfo struct {
	AppID     string `json:"appId"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

// WechatJsapiPayInfo 解析 wechat_jsapi 下单响应中的 pay_info。
func (r *OrderCreateResp) WechatJsapiPayInfo() (*WechatJsapiPayInfo, error) {
	if r == nil || r.PayInfo == nil {
		return nil, fmt.Errorf("sioyun: pay_info is empty")
	}
	data, err := json.Marshal(r.PayInfo)
	if err != nil {
		return nil, fmt.Errorf("sioyun: marshal pay_info: %w", err)
	}
	var info WechatJsapiPayInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("sioyun: unmarshal pay_info: %w", err)
	}
	if info.Package == "" {
		return nil, fmt.Errorf("sioyun: pay_info.package is required for wechat_jsapi")
	}
	return &info, nil
}

// WechatH5PayInfo 解析 wechat_h5 下单响应中的 pay_info。
func (r *OrderCreateResp) WechatH5PayInfo() (string, error) {
	return r.payInfoString("h5_url", "wechat_h5")
}

// WechatNativePayInfo 解析 wechat_native 下单响应中的 pay_info。
func (r *OrderCreateResp) WechatNativePayInfo() (string, error) {
	return r.payInfoString("code_url", "wechat_native")
}

// WechatAppPayInfo 微信 APP 调起支付参数。
type WechatAppPayInfo struct {
	AppID        string `json:"appId"`
	PartnerID    string `json:"partnerId"`
	PrepayID     string `json:"prepayId"`
	PackageValue string `json:"packageValue"`
	NonceStr     string `json:"nonceStr"`
	TimeStamp    string `json:"timeStamp"`
	Sign         string `json:"sign"`
}

// WechatAppPayInfo 解析 wechat_app 下单响应中的 pay_info。
func (r *OrderCreateResp) WechatAppPayInfo() (*WechatAppPayInfo, error) {
	if r == nil || r.PayInfo == nil {
		return nil, fmt.Errorf("sioyun: pay_info is empty")
	}
	data, err := json.Marshal(r.PayInfo)
	if err != nil {
		return nil, fmt.Errorf("sioyun: marshal pay_info: %w", err)
	}
	var info WechatAppPayInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("sioyun: unmarshal pay_info: %w", err)
	}
	if info.PrepayID == "" {
		return nil, fmt.Errorf("sioyun: pay_info.prepayId is required for wechat_app")
	}
	return &info, nil
}

// AlipayQrPayInfo 解析 alipay_qr 下单响应中的 pay_info。
func (r *OrderCreateResp) AlipayQrPayInfo() (string, error) {
	return r.payInfoString("qr_code", "alipay_qr")
}

// AlipayH5PayInfo 解析 alipay_h5 下单响应中的 pay_info。
func (r *OrderCreateResp) AlipayH5PayInfo() (string, error) {
	return r.payInfoString("h5_url", "alipay_h5")
}

// AlipayAppPayInfo 解析 alipay_app 下单响应中的 pay_info。
func (r *OrderCreateResp) AlipayAppPayInfo() (string, error) {
	return r.payInfoString("order_string", "alipay_app")
}

// ── 银联支付 pay_info 解析 ──────────────────────────────────────────────────

// UnionPayQrPayInfo 解析 unionpay_qr 下单响应中的 pay_info (C扫B 二维码)。
func (r *OrderCreateResp) UnionPayQrPayInfo() (string, error) {
	return r.payInfoString("qr_code", "unionpay_qr")
}

// UnionPayMiniPayInfo 小程序支付调起参数。
type UnionPayMiniPayInfo struct {
	MiniPayRequest map[string]interface{} `json:"mini_pay_request"`
	SeqID          string                 `json:"seq_id"`
	MerOrderID     string                 `json:"mer_order_id"`
}

// UnionPayMiniPayInfo 解析 unionpay_mini / unionpay_jsapi 下单响应中的 pay_info。
func (r *OrderCreateResp) UnionPayMiniPayInfo() (*UnionPayMiniPayInfo, error) {
	if r == nil || r.PayInfo == nil {
		return nil, fmt.Errorf("sioyun: pay_info is empty")
	}
	data, err := json.Marshal(r.PayInfo)
	if err != nil {
		return nil, fmt.Errorf("sioyun: marshal pay_info: %w", err)
	}
	var info UnionPayMiniPayInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("sioyun: unmarshal pay_info: %w", err)
	}
	if info.MiniPayRequest == nil {
		return nil, fmt.Errorf("sioyun: pay_info.mini_pay_request is required for unionpay_mini/jsapi")
	}
	return &info, nil
}

func (r *OrderCreateResp) payInfoString(key, method string) (string, error) {
	if r == nil || r.PayInfo == nil {
		return "", fmt.Errorf("sioyun: pay_info is empty")
	}
	v, ok := r.PayInfo[key]
	if !ok {
		return "", fmt.Errorf("sioyun: pay_info.%s is required for %s", key, method)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("sioyun: pay_info.%s is required for %s", key, method)
	}
	return s, nil
}

// Create 创建支付订单。
func (p *PaymentService) Create(ctx context.Context, req OrderCreateReq) (*OrderCreateResp, error) {
	var resp OrderCreateResp
	if err := p.client.do(ctx, "POST", "/pay/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── 查询 ─────────────────────────────────────────────────────────────────

// OrderQueryReq 订单查询请求。
type OrderQueryReq struct {
	OutTradeNo     string `json:"out_trade_no,omitempty"`
	GatewayTradeNo string `json:"gateway_trade_no,omitempty"`
}

// OrderQueryResp 订单查询响应。
type OrderQueryResp struct {
	OutTradeNo     string `json:"out_trade_no"`
	GatewayTradeNo string `json:"gateway_trade_no"`
	Status         string `json:"status"`      // PENDING / SUCCESS / CLOSED / REFUND / REFUND_PART
	PayMethod      string `json:"pay_method"`
	Amount         int64  `json:"amount"`
	PayAmount      int64  `json:"pay_amount"`
	TransactionID  string `json:"transaction_id"`
	PayTime        int64  `json:"pay_time"`
	Attach         string `json:"attach"`
}

// Query 查询订单状态（GET /pay/query/:out_trade_no）。
func (p *PaymentService) Query(ctx context.Context, req OrderQueryReq) (*OrderQueryResp, error) {
	var path string
	switch {
	case req.OutTradeNo != "":
		path = "/pay/query/" + url.PathEscape(req.OutTradeNo)
	case req.GatewayTradeNo != "":
		return nil, fmt.Errorf("sioyun: gateway_trade_no query is not supported yet")
	default:
		return nil, fmt.Errorf("sioyun: out_trade_no is required")
	}

	var resp OrderQueryResp
	if err := p.client.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── 关闭 ─────────────────────────────────────────────────────────────────

// OrderCloseReq 关闭订单请求。
type OrderCloseReq struct {
	OutTradeNo string `json:"out_trade_no"`
}

// OrderCloseResp 关闭订单响应。
type OrderCloseResp struct {
	OutTradeNo string `json:"out_trade_no"`
	Status     string `json:"status"`
}

// Close 关闭未支付的订单。
func (p *PaymentService) Close(ctx context.Context, req OrderCloseReq) (*OrderCloseResp, error) {
	var resp OrderCloseResp
	if err := p.client.do(ctx, "POST", "/pay/close", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── 退款 ─────────────────────────────────────────────────────────────────

// RefundCreateReq 退款申请请求。
type RefundCreateReq struct {
	OutTradeNo   string `json:"out_trade_no"`   // 原订单号
	OutRefundNo  string `json:"out_refund_no"`  // 退款单号
	RefundAmount int64  `json:"refund_amount"`  // 退款金额（分）
	RefundReason string `json:"refund_reason,omitempty"`
}

// RefundCreateResp 退款申请响应。
type RefundCreateResp struct {
	OutRefundNo  string `json:"out_refund_no"`
	RefundID     string `json:"refund_id"`
	RefundAmount int64  `json:"refund_amount"`
	Status       string `json:"status"` // PROCESSING
}

// Refund 申请退款。
func (p *PaymentService) Refund(ctx context.Context, req RefundCreateReq) (*RefundCreateResp, error) {
	var resp RefundCreateResp
	if err := p.client.do(ctx, "POST", "/pay/refund", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── 退款查询 ─────────────────────────────────────────────────────────────

// RefundQueryReq 退款查询请求。
type RefundQueryReq struct {
	OutRefundNo string `json:"out_refund_no"`
}

// RefundQueryResp 退款查询响应。
type RefundQueryResp struct {
	OutRefundNo  string `json:"out_refund_no"`
	RefundID     string `json:"refund_id"`
	OutTradeNo   string `json:"out_trade_no"`
	RefundAmount int64  `json:"refund_amount"`
	Status       string `json:"status"` // PROCESSING / SUCCESS / FAIL
	RefundTime   int64  `json:"refund_time"`
}

// RefundQuery 查询退款状态。
func (p *PaymentService) RefundQuery(ctx context.Context, req RefundQueryReq) (*RefundQueryResp, error) {
	var resp RefundQueryResp
	if err := p.client.do(ctx, "POST", "/pay/refund/query", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
