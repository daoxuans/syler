package i

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// 通过http方式请求Login
type HttpLoginHandler interface {
	HandleLogin(w http.ResponseWriter, r *http.Request)
}

// 通过http方式请求Logout
type HttpLogoutHandler interface {
	HandleLogout(w http.ResponseWriter, r *http.Request)
}

// 通过http方式请求SendCode
type HttpSendCodeHandler interface {
	HandleSendCode(w http.ResponseWriter, r *http.Request)
}

type HttpRootHandler interface {
	HandleRoot(w http.ResponseWriter, r *http.Request)
}

// 通过该接口监听更多的http方法
type ExtraHttpHandler interface {
	AddExtraHttp()
}

var ExtraAuth interface{}

// UTILS for wrap the http error
func ErrorWrap(w http.ResponseWriter) {
	if e := recover(); e != nil {
		stack := debug.Stack()
		log.Printf("panic recovered: %v\nstack trace:\n%s", e, stack)

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
