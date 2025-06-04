package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"daoxuans/syler/component"
	"daoxuans/syler/config"
	"daoxuans/syler/logger"

	toml "github.com/extrame/go-toml-config"
)

func main() {

	configPath := flag.String("config", "./syler.toml", "设置配置文件的路径")
	flag.Parse()
	path := filepath.FromSlash(*configPath)
	if err := toml.Parse(path); err != nil {
		fmt.Printf("解析配置文件失败: %v", err)
		os.Exit(1)
	}

	err := logger.Init(
		*config.LogFile,
		*config.LogLevel,
		*config.LogMaxSize,
		*config.LogMaxBackups,
	)
	if err != nil {
		fmt.Printf("初始化日志失败: %v", err)
		os.Exit(1)
	}

	log := logger.GetLogger()

	component.InitBasic()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	startServices(ctx, &wg)

	handleSignals(cancel)

	wg.Wait()
	log.Println("已安全关闭")
}

func startServices(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		go component.StartHuawei()
		log.Println("Portal 协议服务已启动")

		<-ctx.Done()
		log.Println("正在关闭Portal 协议服务...")
		// 这里可以添加华为服务的清理代码
		time.Sleep(time.Second) // 给清理留出时间
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		go component.StartHttp()
		log.Println("Portal Web服务已启动")

		<-ctx.Done()
		log.Println("正在关闭Portal Web服务...")
		// 这里可以添加HTTP服务的清理代码
		time.Sleep(time.Second) // 给清理留出时间
	}()
}

func handleSignals(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("收到信号: %v, 开始优雅关闭...\n", sig)
		cancel()
	}()
}
