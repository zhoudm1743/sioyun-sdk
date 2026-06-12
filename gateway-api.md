# 开放网关 API 接口文档

**版本**：v1.0  
**根路径**：`https://www.sioyun.com/api/gateway/v1`  
**认证方式**：AK/SK + HMAC-SHA256 签名  
**内容类型**：`application/json`

---

## 目录

1. [通用规范](#1-通用规范)
2. [认证与签名](#2-认证与签名)
3. [短信接口](#3-短信接口)
4. [支付接口](#4-支付接口)
5. [进件接口](#5-进件接口)
6. [应用接口](#6-应用接口)
7. [回调通知接口](#7-回调通知接口)
8. [错误码](#8-错误码)

---

## 1. 通用规范

### 1.1 请求格式

所有请求均为 `POST`（除余额查询为 `GET`），`Content-Type: application/json`。

**必选请求头：**

| 请求头 | 类型 | 说明 |
|--------|------|------|
| `X-Access-Key` | string | 访问标识符（`ak_` 前缀 + 24 位 Base62） |
| `X-Timestamp` | string | Unix 秒级时间戳，与服务器时间偏差 ±300 秒 |
| `X-Nonce` | string | 随机字符串（16-32 位），同一 Nonce 5 分钟内不可重复 |
| `X-Signature` | string | HMAC-SHA256 签名（十六进制） |
| `X-Request-Id` | string | 请求追踪 ID（UUID），可选但强烈推荐 |

### 1.2 统一响应

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

| code | 含义 |
|------|------|
| 0 | 成功 |
| 400 | 参数错误 |
| 401 | 签名验证失败 / AK 无效 |
| 402 | 短信额度不足 |
| 403 | 账户被禁用 / 无权限 |
| 404 | 资源不存在 |
| 429 | 频率限制 |
| 500 | 服务器内部错误 |

### 1.3 签名算法

详见 [第 2 章](#2-认证与签名)。

---

## 2. 认证与签名

### 2.1 AK/SK 获取

在平台用户端的「API 密钥管理」页面（`/app/settings/aksk`）创建密钥对：
- **Access Key**：格式 `ak_xxxxxxxxxxxxxxxxxxxxxxxx`，公开标识符
- **Secret Key**：格式 `sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`，**仅创建时返回一次**

### 2.2 签名计算

```
签名字符串 = HTTP_METHOD + "\n"
           + REQUEST_PATH（不含查询参数）+ "\n"
           + X-Timestamp + "\n"
           + X-Nonce + "\n"
           + SHA256(REQUEST_BODY)

Signature = Hex(HMAC-SHA256(签名字符串, SecretKey))
```

**示例（POST /api/gateway/v1/sms/send）：**

```
请求体：{"phone":"13800138000","template_code":"verify_code","params":{"code":"123456"}}
请求体 SHA256：e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

签名字符串：
POST\n
/api/gateway/v1/sms/send\n
1718150400\n
abc123def456ghi789\n
a3b9c1d2e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1

Signature = HMAC-SHA256(签名字符串, "sk_1234abcd...5678efgh")
```

### 2.3 签名验证失败常见原因

| 现象 | 可能原因 |
|------|---------|
| 401 "签名验证失败" | SecretKey 不正确；签名字符串顺序错误；时间戳偏差超过 300 秒 |
| 401 "Access Key 无效" | AK 不存在或已被禁用 |
| 401 "Nonce 重复" | 同一 Nonce 在 5 分钟内被重复使用 |
| 401 "时间戳过期" | 服务器时间与请求时间相差超过 5 分钟 |

### 2.4 多语言签名示例

**Go：**

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

func Sign(secretKey, method, path, timestamp, nonce, body string) string {
    bodyHash := sha256Hex(body)
    signStr := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", method, path, timestamp, nonce, bodyHash)
    mac := hmac.New(sha256.New, []byte(secretKey))
    mac.Write([]byte(signStr))
    return hex.EncodeToString(mac.Sum(nil))
}

func sha256Hex(s string) string {
    h := sha256.Sum256([]byte(s))
    return hex.EncodeToString(h[:])
}
```

**Python：**

```python
import hmac, hashlib

def sign(secret_key: str, method: str, path: str, timestamp: str, nonce: str, body: str) -> str:
    body_hash = hashlib.sha256(body.encode()).hexdigest()
    sign_str = f"{method}\n{path}\n{timestamp}\n{nonce}\n{body_hash}"
    return hmac.new(secret_key.encode(), sign_str.encode(), hashlib.sha256).hexdigest()
```

**JavaScript：**

```javascript
const crypto = require('crypto');

function sign(secretKey, method, path, timestamp, nonce, body) {
    const bodyHash = crypto.createHash('sha256').update(body).digest('hex');
    const signStr = `${method}\n${path}\n${timestamp}\n${nonce}\n${bodyHash}`;
    return crypto.createHmac('sha256', secretKey).update(signStr).digest('hex');
}
```

---

## 3. 短信接口

### 3.1 发送短信

```
POST /api/gateway/v1/sms/send
```

**前置条件：** 账户需购买短信套餐且有可用额度。

**请求：**

```json
{
  "phone": "13800138000",
  "template_code": "verify_code",
  "params": {
    "code": "123456"
  },
  "signature_name": "西奥科技"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| phone | string | 是 | 目标手机号 |
| template_code | string | 是 | 本地模板编码（平台短信管理后台创建） |
| params | object | 否 | 模板变量键值对 |
| signature_name | string | 否 | 指定签名名称，不传则用模板关联的签名 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "send_id": "submail_send_abc123",
    "fee": 1,
    "balance_remaining": 899
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| send_id | string | 发送流水号（用于追踪送达状态） |
| fee | int | 本次消费条数 |
| balance_remaining | int64 | 账户剩余可用条数 |

**错误响应：**

```json
{
  "code": 402,
  "msg": "短信额度不足，请购买套餐",
  "data": null
}
```

**常见错误：**

| code | 说明 |
|------|------|
| 400 | `template_code` 不存在或未通过审核 |
| 400 | 签名不存在或未通过审核 |
| 402 | 短信额度不足 |
| 429 | 发送频率超限（100 次/分钟） |
| 500 | 供应商发送失败（额度已回滚） |

### 3.2 查询短信余额

```
GET /api/gateway/v1/sms/balance
```

**请求：** 无 Body（GET 请求时签名字符串中 body 为空字符串）。

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total_remaining": 1500,
    "packages": [
      {
        "id": "pkg_001",
        "package_name": "体验包 100条",
        "total": 100,
        "remaining": 80,
        "expired_at": 1720000000
      },
      {
        "id": "pkg_002",
        "package_name": "标准包 1000条",
        "total": 1000,
        "remaining": 1000,
        "expired_at": 0
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| total_remaining | int64 | 所有有效套餐的剩余条数总和 |
| packages | array | 有效套餐明细 |
| packages[].id | string | 套餐购买记录 ID |
| packages[].package_name | string | 套餐名称 |
| packages[].total | int64 | 总条数 |
| packages[].remaining | int64 | 剩余条数 |
| packages[].expired_at | int64 | 到期时间（Unix 秒，0=永久） |

---

## 4. 支付接口

### 4.1 统一下单

```
POST /api/gateway/v1/pay/create
```

**前置条件：** 已完成微信或支付宝进件，拥有子商户号。

**请求：**

```json
{
  "out_trade_no": "ORDER20260612001",
  "amount": 100,
  "pay_method": "wechat_jsapi",
  "description": "购买短信套餐-标准包",
  "notify_url": "https://partner.example.com/callback/payment",
  "openid": "oUpF8uMuAJO_M2pxb1Q9zNjWeS6o",
  "sub_mchid": "1600000001"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| out_trade_no | string | 是 | 商户订单号（唯一，建议长度 6-32 位） |
| amount | int64 | 是 | 支付金额（**分**，100 = 1 元） |
| pay_method | string | 是 | 支付方式（见下表） |
| description | string | 是 | 商品描述（会展示给用户） |
| notify_url | string | 是 | **支付结果回调地址**（网关收到支付平台回调后转发至此） |
| openid | string | 条件 | 微信用户 openid（jsapi/mini 必填） |
| sub_mchid | string | 否 | 指定子商户号（不传使用最近一个进件成功的商户） |
| attach | string | 否 | 附加数据（回调时原样返回，最长 127） |
| expire_minutes | int | 否 | 订单过期分钟数（默认 15，最大 120） |

**支付方式 `pay_method`：**

| 值 | 说明 | 需传字段 |
|----|------|---------|
| `wechat_jsapi` | 微信 JSAPI（公众号/小程序支付） | openid |
| `wechat_h5` | 微信 H5 支付 | - |
| `wechat_native` | 微信 Native（扫码） | - |
| `wechat_app` | 微信 APP 支付 | - |
| `alipay_qr` | 支付宝扫码（当面付） | - |
| `alipay_h5` | 支付宝手机网页 | - |
| `alipay_app` | 支付宝 APP 支付 | - |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "out_trade_no": "ORDER20260612001",
    "gateway_trade_no": "GATEWAY20260612001",
    "pay_method": "wechat_jsapi",
    "amount": 100,
    "pay_info": {
      "appId": "wx374a8c93ace959ea",
      "timeStamp": "1718150400",
      "nonceStr": "abc123def456",
      "package": "prepay_id=wx123456789",
      "signType": "RSA",
      "paySign": "a1b2c3d4e5f6..."
    }
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| out_trade_no | string | 商户订单号（原样返回） |
| gateway_trade_no | string | 网关内部流水号 |
| pay_method | string | 支付方式 |
| amount | int64 | 支付金额（分） |
| pay_info | object | 调起支付的参数（结构根据 pay_method 不同） |

**`pay_info` 结构说明：**

- **wechat_jsapi**：appId, timeStamp, nonceStr, package, signType, paySign（用于 `wx.requestPayment`）
- **wechat_h5**：h5_url（直接跳转此 URL 发起支付）
- **wechat_native**：code_url（生成二维码内容）
- **wechat_app**：appId, partnerId, prepayId, packageValue, nonceStr, timeStamp, sign（用于 APP 调起）
- **alipay_qr**：qr_code（二维码内容）
- **alipay_h5**：h5_url（直接跳转）
- **alipay_app**：order_string（传给 APP SDK）

**常见错误：**

| code | 说明 |
|------|------|
| 400 | `out_trade_no` 重复 |
| 400 | `pay_method` 不支持 |
| 400 | `openid` 缺失（jsapi/mini 支付必填） |
| 404 | 未找到可用的子商户号（请先完成进件） |
| 500 | 支付平台调用失败 |

### 4.2 查询订单

```
POST /api/gateway/v1/pay/query
```

**请求：**

```json
{
  "out_trade_no": "ORDER20260612001"
}
```

或

```json
{
  "gateway_trade_no": "GATEWAY20260612001"
}
```

两个字段二选一，都传则以 `gateway_trade_no` 为准。

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "out_trade_no": "ORDER20260612001",
    "gateway_trade_no": "GATEWAY20260612001",
    "status": "SUCCESS",
    "pay_method": "wechat_jsapi",
    "amount": 100,
    "pay_amount": 100,
    "transaction_id": "420000123456789",
    "pay_time": 1718150500,
    "attach": ""
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | PENDING / SUCCESS / CLOSED / REFUND / REFUND_PART |
| transaction_id | string | 支付平台交易流水号 |
| pay_amount | int64 | 实际支付金额（分） |
| pay_time | int64 | 支付完成时间（Unix 秒，PENDING 时为 0） |

### 4.3 关闭订单

```
POST /api/gateway/v1/pay/close
```

**请求：**

```json
{
  "out_trade_no": "ORDER20260612001"
}
```

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "out_trade_no": "ORDER20260612001",
    "status": "CLOSED"
  }
}
```

> **注意：** 仅 PENDING 状态的订单可以关闭。已支付订单请走退款接口。

### 4.4 申请退款

```
POST /api/gateway/v1/pay/refund
```

**请求：**

```json
{
  "out_trade_no": "ORDER20260612001",
  "out_refund_no": "REFUND20260612001",
  "refund_amount": 50,
  "refund_reason": "用户申请退款"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| out_trade_no | string | 是 | 原支付订单号 |
| out_refund_no | string | 是 | 退款单号（唯一） |
| refund_amount | int64 | 是 | 退款金额（分，≤ 原订单金额） |
| refund_reason | string | 否 | 退款原因 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "out_refund_no": "REFUND20260612001",
    "refund_id": "503000123456789",
    "refund_amount": 50,
    "status": "PROCESSING"
  }
}
```

> **注意：** 退款是异步的。退款成功后，网关会通过 HTTP 回调 + WebSocket 推送 `payment.refund` 事件通知。

### 4.5 查询退款

```
POST /api/gateway/v1/pay/refund/query
```

**请求：**

```json
{
  "out_refund_no": "REFUND20260612001"
}
```

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "out_refund_no": "REFUND20260612001",
    "refund_id": "503000123456789",
    "out_trade_no": "ORDER20260612001",
    "refund_amount": 50,
    "status": "SUCCESS",
    "refund_time": 1718151000
  }
}
```

---

## 5. 进件接口

### 5.1 提交进件申请

```
POST /api/gateway/v1/partner/apply
```

**请求（微信）：**

```json
{
  "channel": "wechat",
  "merchant_name": "测试商户",
  "subject_type": "ENTERPRISE",
  "notify_url": "https://partner.example.com/callback/applyment",
  "form_data": {
    "contact_info": {
      "contact_name": "张三",
      "contact_id_number": "320101199001011234",
      "mobile_phone": "13800138000",
      "contact_email": "zhangsan@example.com"
    },
    "subject_info": {
      "subject_type": "ENTERPRISE",
      "business_license_copy": "https://example.com/images/license.jpg",
      "business_license_number": "91310000MA1XXXXX",
      "merchant_name": "上海测试科技有限公司",
      "legal_person": "李四"
    },
    "business_info": {
      "merchant_shortname": "测试商户",
      "service_phone": "021-12345678",
      "sales_scenes_type": ["SALES_SCENES_OFFLINE"],
      "settlement_id": "91310000MA1XXXXX"
    },
    "settlement_info": {
      "settlement_id": "91310000MA1XXXXX",
      "qualification_type": "统一社会信用代码"
    },
    "bank_account_info": {
      "account_type": "ACCOUNT_TYPE_PRIVATE",
      "account_bank": "工商银行",
      "bank_address_code": "102290000316",
      "bank_name": "中国工商银行上海南京东路支行",
      "account_number": "6222021001000000000",
      "account_name": "张三"
    }
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel | string | 是 | `wechat` 或 `alipay` |
| merchant_name | string | 是 | 商户简称（会展示给用户） |
| subject_type | string | 是 | 主体类型（微信：ENTERPRISE/INDIVIDUAL/... 支付宝：见支付宝文档） |
| notify_url | string | 否 | 进件审批结果回调地址 |
| form_data | object | 是 | 进件表单数据（结构按渠道不同，见下方说明） |

**form_data 说明：**

- **微信进件**：遵循微信官方申请单结构，包含 contact_info、subject_info、business_info、settlement_info、bank_account_info 五大区块。图片字段需传可访问的 URL（网关会自动上传到微信）。
- **支付宝进件**：遵循支付宝 `ant.merchant.expand.indirect.zft.create` 接口参数结构。

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "apply_id": "apply_abc123",
    "applyment_id": 123456789,
    "channel": "wechat",
    "status": "submitted",
    "sign_url": "https://pay.weixin.qq.com/...",
    "submitted_at": 1718150400
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| apply_id | string | 平台申请单 ID（用于查询） |
| applyment_id | int64 | 支付平台申请单号 |
| status | string | submitted / signing / rejected / finished |
| sign_url | string | 签约链接（待签约状态时有值） |
| submitted_at | int64 | 提交时间 |

### 5.2 查询进件状态

```
GET /api/gateway/v1/partner/query/:apply_id
```

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "apply_id": "apply_abc123",
    "applyment_id": 123456789,
    "channel": "wechat",
    "status": "finished",
    "applyment_state": "APPLYMENT_STATE_FINISHED",
    "applyment_state_msg": "签约完成",
    "sub_mchid": "1600000001",
    "sign_url": "",
    "audit_detail": [],
    "submitted_at": 1718150400,
    "finished_at": 1718236800
  }
}
```

**status 字段含义：**

| status | 说明 |
|--------|------|
| draft | 草稿（未提交） |
| submitted | 已提交，审核中 |
| signing | 待签约（需法人确认） |
| rejected | 已驳回（查看 audit_detail 了解原因） |
| finished | 完成（已获得子商户号） |
| canceled | 已作废 |

---

## 6. 应用接口

### 6.1 查询已订阅应用

```
GET /api/gateway/v1/app/subscriptions
```

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "id": "sub_001",
      "product_id": "prod_001",
      "product_name": "短信网关",
      "product_logo": "https://example.com/logo.png",
      "version": "v2.0",
      "price_type": 2,
      "amount": 29900,
      "status": 1,
      "start_at": 1718150400,
      "expire_at": 1749686400
    }
  ]
}
```

### 6.2 查询账户信息

```
GET /api/gateway/v1/app/profile
```

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": "user_001",
    "username": "partner_company",
    "nickname": "合作伙伴公司",
    "email": "partner@example.com",
    "wallet_balance": 50000,
    "sms_remaining": 1500,
    "wechat_merchants": [
      {
        "sub_mchid": "1600000001",
        "merchant_name": "测试商户",
        "status": "finished"
      }
    ],
    "alipay_merchants": [
      {
        "smid": "2088000000000001",
        "merchant_name": "测试商户",
        "status": "finished"
      }
    ]
  }
}
```

---

## 7. 回调通知接口

> 回调是网关**主动推送**给合作伙伴的反向接口。合作伙伴需在自己的服务器上实现回调接收端点。

### 7.1 回调协议

网关向合作伙伴 `notify_url` 发送 POST 请求：

**请求头：**

| 请求头 | 说明 |
|--------|------|
| Content-Type | application/json |
| X-Gateway-Signature | HMAC-SHA256 签名（同请求签名算法，用 SecretKey 计算） |
| X-Gateway-Timestamp | 事件发生时间（Unix 秒） |
| X-Gateway-Event | 事件类型（如 payment.success） |

**请求体：**

```json
{
  "event": "payment.success",
  "gateway_trade_no": "GATEWAY20260612001",
  "out_trade_no": "ORDER20260612001",
  "event_time": 1718150500,
  "data": {
    "status": "SUCCESS",
    "pay_amount": 100,
    "transaction_id": "420000123456789",
    "pay_time": 1718150500
  }
}
```

**合作伙伴必须返回：**

```
HTTP 200
Content-Type: text/plain

success
```

> 任何非 200 状态码或 body 不包含 "success" 均视为投递失败，网关会重试最多 10 次。

### 7.2 验签（合作伙伴侧）

```go
// 从请求头取 signature，用本地 SecretKey 计算期望值比对
func VerifySignature(secretKey, signature, body string) bool {
    h := sha256.Sum256([]byte(body))
    bodyHash := hex.EncodeToString(h[:])
    
    signStr := event + "\n" + gatewayTradeNo + "\n" + eventTime + "\n" + bodyHash
    mac := hmac.New(sha256.New, []byte(secretKey))
    mac.Write([]byte(signStr))
    expectedSign := hex.EncodeToString(mac.Sum(nil))
    
    return hmac.Equal([]byte(signature), []byte(expectedSign))
}
```

### 7.3 事件类型与 data 结构

#### `payment.success` — 支付成功

```json
{
  "status": "SUCCESS",
  "pay_amount": 100,
  "transaction_id": "420000123456789",
  "pay_time": 1718150500
}
```

#### `payment.closed` — 支付关闭

```json
{
  "status": "CLOSED",
  "close_reason": "订单超时"
}
```

#### `payment.refund` — 退款到账

```json
{
  "out_refund_no": "REFUND20260612001",
  "refund_id": "503000123456789",
  "refund_amount": 50,
  "refund_status": "SUCCESS",
  "refund_time": 1718151000
}
```

#### `payment.refund.fail` — 退款失败

```json
{
  "out_refund_no": "REFUND20260612001",
  "refund_id": "503000123456789",
  "refund_amount": 50,
  "fail_reason": "账户余额不足"
}
```

#### `sms.delivered` — 短信送达

```json
{
  "send_id": "submail_send_abc123",
  "phone_hash": "a1b2c3d4...",
  "fee": 1,
  "delivered_at": 1718150500
}
```

#### `sms.dropped` — 短信投递失败

```json
{
  "send_id": "submail_send_abc123",
  "phone_hash": "a1b2c3d4...",
  "drop_reason": "号码不存在"
}
```

#### `partner.applyment.approved` — 进件审批通过

```json
{
  "apply_id": "apply_abc123",
  "channel": "wechat",
  "sub_mchid": "1600000001",
  "approved_at": 1718236800
}
```

#### `partner.applyment.rejected` — 进件被驳回

```json
{
  "apply_id": "apply_abc123",
  "channel": "wechat",
  "reject_reason": "营业执照模糊不清",
  "audit_detail": [
    {
      "field": "business_license_copy",
      "field_name": "营业执照",
      "reject_reason": "图片模糊，请重新上传"
    }
  ]
}
```

#### `partner.applyment.signing` — 待签约确认

```json
{
  "apply_id": "apply_abc123",
  "channel": "wechat",
  "sign_url": "https://pay.weixin.qq.com/sign/..."
}
```

### 7.4 重试策略

| 重试次数 | 延迟 |
|---------|------|
| 第 1 次 | 1 分钟 |
| 第 2 次 | 2 分钟 |
| 第 3 次 | 4 分钟 |
| 第 4 次 | 8 分钟 |
| ... | 指数退避 |
| 第 10 次 | ~8.5 小时 |
| 第 11 次 | 放弃，进死信队列 |

### 7.5 回调管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/gateway/v1/notify/status/:gateway_trade_no` | 查询回调投递状态 |
| GET | `/api/gateway/v1/notify/logs?out_trade_no=xxx` | 按订单号查询投递日志 |
| POST | `/api/gateway/v1/notify/retry/:gateway_trade_no` | 手动触发重新投递 |

---

## 8. 错误码

### 8.1 通用错误

| code | msg 示例 | 说明 |
|------|---------|------|
| 400 | 参数校验失败: phone is required | 请求参数不合法 |
| 401 | 签名验证失败 | AK 或签名错误 |
| 401 | Access Key 无效 | AK 不存在或已禁用 |
| 401 | Nonce 重复使用 | 同一 Nonce 5 分钟内重复 |
| 401 | 时间戳已过期 | 与服务器时间差 > 300 秒 |
| 403 | 账户已被禁用 | 账户被管理员冻结 |
| 429 | 请求频率超限 | 超过接口限流阈值 |
| 500 | 服务器内部错误 | 联系技术支持 |

### 8.2 短信错误

| code | msg | 说明 |
|------|-----|------|
| 400 | 模板不可用 | template_code 不存在或审核未通过 |
| 400 | 签名不可用 | 签名不存在或审核未通过 |
| 400 | 手机号格式错误 | |
| 402 | 短信额度不足，请购买套餐 | 账户无可用额度 |
| 429 | 发送频率超限 | 超过 100 次/分钟 或 1000 次/天 |
| 500 | 供应商发送失败 | 上游供应商错误 |

### 8.3 支付错误

| code | msg | 说明 |
|------|-----|------|
| 400 | 订单号重复 | out_trade_no 已存在 |
| 400 | 支付方式不支持 | pay_method 不在支持列表中 |
| 400 | openid 不能为空 | JSAPI/小程序支付必填 |
| 404 | 未找到可用商户号 | 未完成进件或进件未通过 |
| 404 | 订单不存在 | 查询/关闭/退款时订单号错误 |

### 8.4 进件错误

| code | msg | 说明 |
|------|-----|------|
| 400 | 进件表单数据为空 | form_data 必填 |
| 400 | 缺少必填区块: subject_info | 表单结构不完整 |
| 500 | 微信进件失败: 商户信息校验不通过 | 支付平台返回错误 |

---

*文档结束*
