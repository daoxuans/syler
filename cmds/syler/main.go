package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"daoxuans/syler/component"
	"daoxuans/syler/logger"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	// Load configuration
	viper.SetConfigName("syler")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/syler/")
	viper.AddConfigPath("$HOME/.syler")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file: %s\n", err)
		os.Exit(1)
	}

	// Initialize logger
	err := logger.Init(
		viper.GetString("logging.file"),
		viper.GetString("logging.level"),
		viper.GetInt("logging.max_size"),
		viper.GetInt("logging.max_backups"),
	)
	if err != nil {
		fmt.Printf("Error initializing logger: %s\n", err)
		os.Exit(1)
	}

	log := logger.GetLogger()

	// Initialize basic components
	component.InitAuthenticator()

	// Handle graceful shutdown
	shutdown := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.WithFields(logrus.Fields{
			"signal": "SIGTERM/SIGINT",
		}).Info("Shutting down server")
		close(shutdown)
	}()

	// Start portal server
	go component.StartHuawei()

	// Start HTTP server
	go component.StartHttp()

	<-shutdown
}
