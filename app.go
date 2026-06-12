package sioyun

import "context"

// AppService 应用服务（查询已订阅应用与账户信息）。
type AppService struct {
	client *Client
}

// ── 订阅列表 ─────────────────────────────────────────────────────────────

// SubscriptionInfo 订阅信息。
type SubscriptionInfo struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	ProductLogo string `json:"product_logo"`
	Version     string `json:"version"`
	PriceType   int8   `json:"price_type"`
	Amount      int64  `json:"amount"`
	Status      int8   `json:"status"` // 1-有效 0-已过期 2-已取消
	StartAt     int64  `json:"start_at"`
	ExpireAt    int64  `json:"expire_at"`
}

// Subscriptions 查询已订阅应用。
func (a *AppService) Subscriptions(ctx context.Context) ([]SubscriptionInfo, error) {
	var resp []SubscriptionInfo
	if err := a.client.do(ctx, "GET", "/app/subscriptions", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ── 账户信息 ─────────────────────────────────────────────────────────────

// ProfileResp 账户信息响应。
type ProfileResp struct {
	UserID          string          `json:"user_id"`
	Username        string          `json:"username"`
	Nickname        string          `json:"nickname"`
	Email           string          `json:"email"`
	WalletBalance   int64           `json:"wallet_balance"`
	SMSRemaining    int64           `json:"sms_remaining"`
	WechatMerchants []MerchantBrief `json:"wechat_merchants"`
	AlipayMerchants []MerchantBrief `json:"alipay_merchants"`
}

// MerchantBrief 商户简要信息。
type MerchantBrief struct {
	SubMchID     string `json:"sub_mchid"`
	SMID         string `json:"smid"`
	MerchantName string `json:"merchant_name"`
	Status       string `json:"status"`
}

// Profile 查询当前账户信息。
func (a *AppService) Profile(ctx context.Context) (*ProfileResp, error) {
	var resp ProfileResp
	if err := a.client.do(ctx, "GET", "/app/profile", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
