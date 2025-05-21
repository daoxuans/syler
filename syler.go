package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"

	"daoxuans/syler/component"
	"daoxuans/syler/config"

	toml "github.com/extrame/go-toml-config"
)

func main() {
	//http server
	//radius avp server
	//radius accounting server

	// go func() {
	// 	res, err := v1.Challenge(net.IPv4(192, 168, 10, 254), *config.PortalSecret, net.IPv4(192, 168, 56, 2))
	// }()

	runtime.GOMAXPROCS(runtime.NumCPU())

	path := flag.String("config", "./syler.toml", "设置配置文件的路径")
	component.InitBasic()
	flag.Parse()
	*path = filepath.FromSlash(*path)
	if err := toml.Parse(*path); err == nil {
		if config.IsValid() {
			component.InitLogger()
			go component.StartHuawei()
			if *config.RadiusEnable {
				go component.StartRadiusAuth()
				go component.StartRadiusAcc()
			}
			component.StartHttp()
		}
	} else {
		fmt.Println(err)
	}
}
