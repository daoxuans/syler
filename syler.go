package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"daoxuans/syler/component"
	"daoxuans/syler/config"

	toml "github.com/extrame/go-toml-config"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	configPath := flag.String("config", "./syler.toml", "设置配置文件的路径")
	flag.Parse()

	if err := initialize(*configPath); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	startServices(ctx, &wg)

	handleSignals(cancel)

	wg.Wait()
	log.Println("已安全关闭")
}

func initialize(configPath string) error {

	configPath = filepath.FromSlash(configPath)
	if err := toml.Parse(configPath); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	if !config.IsValid() {
		return fmt.Errorf("配置验证失败")
	}

	component.InitLogger()

	component.InitBasic()

	return nil
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
