package sioyun_test

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	sioyun "github.com/zhoudm1743/sioyun-sdk"
)

func ExampleWSClient() {
	client, err := sioyun.New(sioyun.Config{
		BaseURL:   "https://www.sioyun.com/api/gateway/v1",
		AccessKey: os.Getenv("SIOYUN_AK"),
		SecretKey: os.Getenv("SIOYUN_SK"),
	})
	if err != nil {
		log.Fatal(err)
	}
	_ = client // HTTP 客户端

	// 创建独立的 WS 客户端
	ws := sioyun.NewWSClient(sioyun.Config{
		BaseURL:   "https://www.sioyun.com/api/gateway/v1",
		AccessKey: os.Getenv("SIOYUN_AK"),
		SecretKey: os.Getenv("SIOYUN_SK"),
	})

	// 注册事件处理器
	ws.On("payment.success", func(event sioyun.GatewayEvent) {
		fmt.Printf("收到支付成功通知: %+v\n", event.Data)
	})

	ws.On("payment.refund", func(event sioyun.GatewayEvent) {
		fmt.Printf("收到退款通知: %+v\n", event.Data)
	})

	ws.On("payment.*", func(event sioyun.GatewayEvent) {
		fmt.Printf("收到支付相关事件: %s\n", event.Event)
	})

	// 建立连接
	if err := ws.Connect(nil); err != nil {
		log.Fatal(err)
	}

	// 订阅频道
	ws.Subscribe("payment.*")
	ws.Subscribe("sms.delivered")

	// 优雅退出
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	ws.Close()
	time.Sleep(time.Second)
}
