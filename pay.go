package sioyun

import "context"

// PaymentService 支付服务。
type PaymentService struct {
	client *Client
}

// ── 下单 ─────────────────────────────────────────────────────────────────

// OrderCreateReq 支付下单请求。
type OrderCreateReq struct {
	OutTradeNo    string `json:"out_trade_no"`              // 必填：商户订单号（唯一）
	Amount        int64  `json:"amount"`                    // 必填：金额（分）
	PayMethod     string `json:"pay_method"`                // 必填：wechat_jsapi / wechat_h5 / wechat_native / alipay_qr / alipay_h5 / wechat_app / alipay_app
	Description   string `json:"description"`               // 必填：商品描述
	NotifyURL     string `json:"notify_url"`                // 必填：支付结果回调地址
	OpenID        string `json:"openid,omitempty"`          // 条件：微信 jsapi 支付必填
	SubMchID      string `json:"sub_mchid,omitempty"`       // 指定子商户号
	Attach        string `json:"attach,omitempty"`          // 附加数据（回调原样返回）
	ExpireMinutes int    `json:"expire_minutes,omitempty"`  // 过期分钟数
}

// OrderCreateResp 支付下单响应。
type OrderCreateResp struct {
	OutTradeNo     string                 `json:"out_trade_no"`
	GatewayTradeNo string                 `json:"gateway_trade_no"`
	PayMethod      string                 `json:"pay_method"`
	Amount         int64                  `json:"amount"`
	PayInfo        map[string]interface{} `json:"pay_info"`
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

// Query 查询订单状态。
func (p *PaymentService) Query(ctx context.Context, req OrderQueryReq) (*OrderQueryResp, error) {
	var resp OrderQueryResp
	if err := p.client.do(ctx, "POST", "/pay/query", req, &resp); err != nil {
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
