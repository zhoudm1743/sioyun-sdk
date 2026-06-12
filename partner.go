package sioyun

import "context"

// PartnerService 进件服务（微信/支付宝特约商户入驻）。
type PartnerService struct {
	client *Client
}

// ── 提交 ─────────────────────────────────────────────────────────────────

// ApplymentSubmitReq 进件提交请求。
type ApplymentSubmitReq struct {
	Channel      string                 `json:"channel"`       // wechat | alipay
	MerchantName string                 `json:"merchant_name"` // 商户简称
	SubjectType  string                 `json:"subject_type"`  // 主体类型（微信: ENTERPRISE/INDIVIDUAL/...）
	NotifyURL    string                 `json:"notify_url,omitempty"`
	FormData     map[string]interface{} `json:"form_data"`     // 进件表单（结构按 channel 不同）
}

// ApplymentSubmitResp 进件提交响应。
type ApplymentSubmitResp struct {
	ApplyID      string `json:"apply_id"`
	ApplymentID  int64  `json:"applyment_id"`
	Channel      string `json:"channel"`
	Status       string `json:"status"` // submitted / signing / rejected / finished
	SignURL      string `json:"sign_url"`
	SubmittedAt  int64  `json:"submitted_at"`
}

// Submit 提交进件申请。
func (p *PartnerService) Submit(ctx context.Context, req ApplymentSubmitReq) (*ApplymentSubmitResp, error) {
	var resp ApplymentSubmitResp
	if err := p.client.do(ctx, "POST", "/partner/apply", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── 查询 ─────────────────────────────────────────────────────────────────

// ApplymentQueryResp 进件查询响应。
type ApplymentQueryResp struct {
	ApplyID           string        `json:"apply_id"`
	ApplymentID       int64         `json:"applyment_id"`
	Channel           string        `json:"channel"`
	Status            string        `json:"status"` // draft/submitted/signing/rejected/finished/canceled
	ApplymentState    string        `json:"applyment_state"`
	ApplymentStateMsg string        `json:"applyment_state_msg"`
	SubMchID          string        `json:"sub_mchid"` // 微信子商户号
	SMID              string        `json:"smid"`       // 支付宝 smid
	SignURL           string        `json:"sign_url"`
	AuditDetail       []AuditDetail `json:"audit_detail"`
	SubmittedAt       int64         `json:"submitted_at"`
	FinishedAt        int64         `json:"finished_at"`
}

// AuditDetail 审核驳回详情。
type AuditDetail struct {
	Field        string `json:"field"`
	FieldName    string `json:"field_name"`
	RejectReason string `json:"reject_reason"`
}

// Query 查询进件状态。
func (p *PartnerService) Query(ctx context.Context, applyID string) (*ApplymentQueryResp, error) {
	var resp ApplymentQueryResp
	if err := p.client.do(ctx, "GET", "/partner/query/"+applyID, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
