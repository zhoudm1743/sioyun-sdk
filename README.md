# sioyun-sdk

西奥开放网关 Go SDK，提供短信、支付、进件、应用查询等能力，支持 AK/SK 签名认证、WebSocket 实时推送及回调验签。

## 安装

```bash
go get github.com/zhoudm1743/sioyun-sdk@latest
```

**依赖**：仅 `gorilla/websocket`（WebSocket 客户端），核心 HTTP 客户端零外部依赖。

> 最低 Go 版本：1.22

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    sioyun "github.com/zhoudm1743/sioyun-sdk"
)

func main() {
    // 1. 创建客户端（自动验证连通性）
    client, err := sioyun.New(sioyun.Config{
        BaseURL:   "https://www.sioyun.com/api/gateway/v1",
        AccessKey: "ak_xxxxxxxxxxxxxxxxxxxxxxxx",
        SecretKey: "sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
        Timeout:   30,
    })
    if err != nil {
        panic(fmt.Sprintf("SDK 初始化失败: %v", err))
    }

    ctx := context.Background()

    // 2. 发送验证码
    result, err := client.SMS().Send(ctx, sioyun.SmsSendReq{
        Phone:        "13800138000",
        TemplateCode: "verify_code",
        Params:       map[string]string{"code": "123456"},
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("发送成功, send_id=%s, 消费 %d 条\n", result.SendID, result.Fee)

    // 3. 创建支付订单
    order, err := client.Pay().Create(ctx, sioyun.OrderCreateReq{
        OutTradeNo:  "ORDER20260612001",
        Amount:      100,
        PayMethod:   "wechat_jsapi",
        Description: "测试商品",
        NotifyURL:   "https://partner.example.com/callback",
        OpenID:      "oUpF8uMuAJO_M2pxb1Q9zNjWeS6o",
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("下单成功, prepay_id=%s\n", order.PayInfo["package"])
}
```

### 签名算法

```
签名字符串 = METHOD + "\n" + PATH + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + SHA256(BODY)
HMAC key   = SHA256(SecretKey)
签名       = Hex(HMAC-SHA256(签名字符串, HMAC_key))
```

客户端自动处理签名，调用方无需关心。密钥仅在平台上创建时返回一次明文，请妥善保管。

## API 概览

### 客户端配置

```go
client, err := sioyun.New(sioyun.Config{
    BaseURL:        "https://www.sioyun.com/api/gateway/v1",
    AccessKey:      "ak_xxx",
    SecretKey:      "sk_xxx",
    Timeout:        30,       // 秒，默认 30
    CallbackSecret: "sk_xxx", // 回调验签密钥，默认等于 SecretKey
})
```

| 服务 | 获取方式 | 提供能力 |
|------|----------|----------|
| 短信 | `client.SMS()` | 发送短信、查询余额 |
| 支付 | `client.Pay()` | 下单、查询、关闭、退款、退款查询 |
| 进件 | `client.Partner()` | 提交进件、查询状态 |
| 应用 | `client.App()` | 查询订阅列表、账户信息 |

---

### 短信服务 (`client.SMS()`)

**发送短信**

```go
resp, err := client.SMS().Send(ctx, sioyun.SmsSendReq{
    Phone:        "13800138000",
    TemplateCode: "verify_code",
    Params:       map[string]string{"code": "123456"},
})
// resp.SendID  - 发送流水号
// resp.Fee     - 本次消费条数
// resp.BalanceRemaining - 剩余可用条数
```

**查询余额**

```go
resp, err := client.SMS().Balance(ctx)
// resp.TotalRemaining - 总剩余条数
// resp.Packages       - 有效套餐明细
```

---

### 支付服务 (`client.Pay()`)

**下单**

```go
resp, err := client.Pay().Create(ctx, sioyun.OrderCreateReq{
    OutTradeNo:  "ORDER001",
    Amount:      100,          // 金额（分）
    PayMethod:   "wechat_jsapi",
    Description: "商品描述",
    NotifyURL:   "https://example.com/callback",
    OpenID:      "oUpF8uMu...", // JSAPI 必填
})
// resp.GatewayTradeNo - 网关流水号
// resp.PayInfo        - 调起支付所需参数
```

支持的支付方式：`wechat_jsapi` | `wechat_h5` | `wechat_native` | `wechat_app` | `alipay_qr` | `alipay_h5` | `alipay_app`

**查询订单**

```go
resp, err := client.Pay().Query(ctx, sioyun.OrderQueryReq{
    OutTradeNo: "ORDER001",
})
// resp.Status - PENDING / SUCCESS / CLOSED / REFUND / REFUND_PART
```

**关闭订单**

```go
resp, err := client.Pay().Close(ctx, sioyun.OrderCloseReq{
    OutTradeNo: "ORDER001",
})
```

**退款**

```go
resp, err := client.Pay().Refund(ctx, sioyun.RefundCreateReq{
    OutTradeNo:   "ORDER001",
    OutRefundNo:  "REFUND001",
    RefundAmount: 50,
})
// resp.Status - PROCESSING
```

**查询退款**

```go
resp, err := client.Pay().RefundQuery(ctx, sioyun.RefundQueryReq{
    OutRefundNo: "REFUND001",
})
// resp.Status - PROCESSING / SUCCESS / FAIL
```

---

### 进件服务 (`client.Partner()`)

**提交进件**

```go
resp, err := client.Partner().Submit(ctx, sioyun.ApplymentSubmitReq{
    Channel:      "wechat",
    MerchantName: "测试商户",
    SubjectType:  "ENTERPRISE",
    FormData: map[string]interface{}{
        "contact_info":      map[string]interface{}{"contact_name": "张三", ...},
        "subject_info":      map[string]interface{}{...},
        "business_info":     map[string]interface{}{...},
        "settlement_info":   map[string]interface{}{...},
        "bank_account_info": map[string]interface{}{...},
    },
})
// resp.ApplyID  - 申请单 ID
// resp.Status   - submitted / signing / rejected / finished
```

**查询进件状态**

```go
resp, err := client.Partner().Query(ctx, "apply_xxx")
// resp.Status       - draft / submitted / signing / rejected / finished / canceled
// resp.SubMchID     - 微信子商户号（进件完成后返回）
// resp.AuditDetail  - 审核驳回详情
```

---

### 应用服务 (`client.App()`)

**查询已订阅应用**

```go
subs, err := client.App().Subscriptions(ctx)
// []sioyun.SubscriptionInfo
```

**查询账户信息**

```go
resp, err := client.App().Profile(ctx)
// resp.WalletBalance - 钱包余额
// resp.SMSRemaining  - 短信剩余额度
// resp.WechatMerchants / resp.AlipayMerchants - 进件商户列表
```

---

## 错误处理

所有 API 方法在 HTTP 状态非 200 或响应 `code != 0` 时返回 `*sioyun.APIError`。

```go
resp, err := client.SMS().Send(ctx, req)
if err != nil {
    if sioyun.IsInsufficientFunds(err) {
        // code=402，短信额度不足
    }
    if sioyun.IsRateLimited(err) {
        // code=429，频率超限
    }
    if apiErr, ok := err.(*sioyun.APIError); ok {
        fmt.Printf("code=%d, msg=%s\n", apiErr.Code, apiErr.Msg)
    }
}
```

| 错误码常量 | 值 | 含义 |
|-----------|-----|------|
| `ErrCodeBadRequest` | 400 | 参数错误 |
| `ErrCodeUnauthorized` | 401 | 签名验证失败 / AK 无效 |
| `ErrCodeInsufficientFunds` | 402 | 短信额度不足 |
| `ErrCodeForbidden` | 403 | 账户被禁用 |
| `ErrCodeNotFound` | 404 | 资源不存在 |
| `ErrCodeRateLimited` | 429 | 频率限制 |
| `ErrCodeInternalError` | 500 | 服务器内部错误 |

---

## WebSocket 实时推送

```go
ws := sioyun.NewWSClient(sioyun.Config{
    BaseURL:   "https://www.sioyun.com/api/gateway/v1",
    AccessKey: os.Getenv("SIOYUN_AK"),
    SecretKey: os.Getenv("SIOYUN_SK"),
})

// 注册事件处理器（支持通配符）
ws.On("payment.success", func(event sioyun.GatewayEvent) {
    fmt.Printf("支付成功: %+v\n", event.Data)
})
ws.On("payment.*", func(event sioyun.GatewayEvent) {
    fmt.Printf("支付事件: %s\n", event.Event)
})

// 建立连接
ws.Connect(context.Background())

// 订阅频道
ws.Subscribe("payment.*")
ws.Subscribe("sms.delivered")

// 断开连接
defer ws.Close()
```

**特性**：自动断线重连（指数退避 1s→2s→4s→8s→...→60s）、30 秒心跳保活。

---

## 回调验签

网关支付/进件/短信等异步结果通过 HTTP 回调通知合作伙伴，SDK 提供验签工具：

### 方式一：便捷 Handler

```go
http.HandleFunc("/callback/payment", sioyun.CallbackHandler(sioyun.Config{
    SecretKey: "sk_xxx",
}, func(payload *sioyun.CallbackPayload) error {
    fmt.Printf("收到回调: event=%s, data=%+v\n", payload.Event, payload.Data)
    // 更新本地订单状态...
    return nil
}))
```

### 方式二：手动验签

```go
func handleCallback(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)

    payload, _ := sioyun.VerifyAndParseCallback(body, secretKey, "")
    if err := sioyun.VerifySignature(secretKey, payload, r.Header.Get("X-Gateway-Signature")); err != nil {
        http.Error(w, "验签失败", http.StatusForbidden)
        return
    }
    // 处理业务...
    w.Write([]byte("success"))
}
```

---

## 设计原则

- **零外部依赖**（除 `gorilla/websocket` 用于 WS 客户端）
- 单实例复用，线程安全
- 类型安全的请求/响应体
- 自动签名，调用方无需关心签名逻辑
- 初始化时连通性验证

## 相关文档

- [API 接口文档](./gateway-api.md) — 接口详细字段、签名算法、错误码

## License

Apache-2.0
