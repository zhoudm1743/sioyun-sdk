package sioyun

import "context"

// SmsService 短信服务。
type SmsService struct {
	client *Client
}

// ── 发送 ─────────────────────────────────────────────────────────────────

// SmsSendReq 短信发送请求。
type SmsSendReq struct {
	Phone         string            `json:"phone"`                    // 必填：目标手机号
	TemplateCode  string            `json:"template_code"`            // 必填：平台模板编码
	Params        map[string]string `json:"params,omitempty"`         // 模板变量
	SignatureName string            `json:"signature_name,omitempty"` // 指定签名（不传用模板关联的）
}

// SmsSendResp 短信发送响应。
type SmsSendResp struct {
	SendID           string `json:"send_id"`
	Fee              int    `json:"fee"`
	BalanceRemaining int64  `json:"balance_remaining"`
}

// Send 发送模板短信，消费套餐额度。
func (s *SmsService) Send(ctx context.Context, req SmsSendReq) (*SmsSendResp, error) {
	var resp SmsSendResp
	if err := s.client.do(ctx, "POST", "/sms/send", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── 余额 ─────────────────────────────────────────────────────────────────

// SmsBalanceResp 余额查询响应。
type SmsBalanceResp struct {
	TotalRemaining int64            `json:"total_remaining"`
	Packages       []SmsPackageInfo `json:"packages"`
}

// SmsPackageInfo 套餐信息。
type SmsPackageInfo struct {
	ID          string `json:"id"`
	PackageName string `json:"package_name"`
	Total       int64  `json:"total"`
	Remaining   int64  `json:"remaining"`
	ExpiredAt   int64  `json:"expired_at"`
}

// Balance 查询短信余额。
func (s *SmsService) Balance(ctx context.Context) (*SmsBalanceResp, error) {
	var resp SmsBalanceResp
	if err := s.client.do(ctx, "GET", "/sms/balance", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
