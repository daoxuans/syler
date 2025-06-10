package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"syler/internal/logger"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// UTILS for wrap the http error
func ErrorWrap(w http.ResponseWriter) {
	if e := recover(); e != nil {

		if w.Header().Get("Content-Type") != "" {
			return
		}

		errMsg := "服务器内部错误"
		if err, ok := e.(error); ok {
			errMsg = err.Error()
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": errMsg,
			"data": map[string]interface{}{
				"timestamp": time.Now().Format("2006-01-02 15:04:05"),
			},
		})
	}
}

func StartHttp() {

	log := logger.GetLogger()

	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			ErrorWrap(w)
		}()

		AuthHandler.HandleLogin(w, r)
	})
	http.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			ErrorWrap(w)
		}()

		AuthHandler.HandleLogout(w, r)
	})
	http.HandleFunc("/api/sendcode", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			ErrorWrap(w)
		}()

		AuthHandler.HandleSendCode(w, r)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			ErrorWrap(w)
		}()

		AuthHandler.HandleRoot(w, r)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", viper.GetString("http.host"), viper.GetInt("http.port")),
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.WithFields(logrus.Fields{
		"host": viper.GetString("http.host"),
		"port": viper.GetInt("http.port"),
	}).Info("Starting HTTP server")

	if err := server.ListenAndServe(); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to start HTTP server")
	}
}
