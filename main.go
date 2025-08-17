package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"

	"proxy_server/config"
	"proxy_server/log"
	"proxy_server/server"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config_path", "", "配置文件路径")
	flag.Parse()

	fmt.Println(configPath)

	config.Load(configPath)
	log.Init(filepath.Join(config.GetConf().LogDir))
	go gohttp()
}

// main 程序主入口
func main() {
	// 1. 启动服务器
	err := server.Start()
	if err != nil {
		// 启动失败记录致命错误并退出
		log.Fatal("服务器启动失败",
			zap.String("error", err.Error()))
		return
	}

	// 确保程序退出时执行清理工作
	defer func() {
		server.Stop() // 停止服务器
		log.Info("程序正常关闭")
	}()

	// 2. 设置信号监听通道
	signalChan := make(chan os.Signal, 1) // 缓冲大小为1的信号通道

	// 注册要监听的信号：
	// - SIGINT (Ctrl+C中断)
	// - SIGTERM (终止信号)
	// - Kill信号 (强制终止)
	signal.Notify(signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Kill,
	)

	<-signalChan // 当接收到上述任意信号时继续执行
	time.AfterFunc(5*time.Second, func() {
		log.Info("程序强制关闭")
		os.Exit(1)
	})
}
