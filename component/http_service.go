package component

import (
	"fmt"
	"net/http"
	"time"

	"daoxuans/syler/config"
	"daoxuans/syler/i"
	"daoxuans/syler/logger"
)

func StartHttp() {

	log := logger.GetLogger()

	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			i.ErrorWrap(w)
		}()
		if handler, ok := i.ExtraAuth.(i.HttpLoginHandler); ok {
			handler.HandleLogin(w, r)
		} else {
			BASIC_SERVICE.HandleLogin(w, r)
		}
	})
	http.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			i.ErrorWrap(w)
		}()
		if handler, ok := i.ExtraAuth.(i.HttpLogoutHandler); ok {
			handler.HandleLogout(w, r)
		} else {
			BASIC_SERVICE.HandleLogout(w, r)
		}
	})
	http.HandleFunc("/api/sendcode", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			i.ErrorWrap(w)
		}()
		if handler, ok := i.ExtraAuth.(i.HttpSendCodeHandler); ok {
			handler.HandleSendCode(w, r)
		} else {
			BASIC_SERVICE.HandleSendCode(w, r)
		}
	})
	if extrahttp, ok := i.ExtraAuth.(i.ExtraHttpHandler); ok {
		extrahttp.AddExtraHttp()
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			i.ErrorWrap(w)
		}()
		if handler, ok := i.ExtraAuth.(i.HttpRootHandler); ok {
			handler.HandleRoot(w, r)
		} else {
			BASIC_SERVICE.HandleRoot(w, r)
		}
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("localhost:%d", *config.HttpPort),
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Printf("listen http on %d\n", *config.HttpPort)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start HTTP server on port %d: %v", *config.HttpPort, err)
	}
}
