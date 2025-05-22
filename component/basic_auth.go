package component

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"daoxuans/syler/config"
	"daoxuans/syler/i"
)

type AuthInfo struct {
	Name    []byte
	Pwd     []byte
	Mac     net.HardwareAddr
	Timeout uint32 //用户会话超时时间，单位秒
}

type AuthServer struct {
	authing_user map[string]*AuthInfo
	templates    map[string]*template.Template
}

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data,omitempty"`
}

func handleResponse(w http.ResponseWriter, resp Response, template string) {
	if template != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := BASIC_SERVICE.templates[template].Execute(w, resp); err != nil {
			log.Printf("Template execution failed: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.Code)
	json.NewEncoder(w).Encode(resp)
}

var BASIC_SERVICE = new(AuthServer)

func InitBasic() {
	BASIC_SERVICE = &AuthServer{
		authing_user: make(map[string]*AuthInfo),
		templates:    make(map[string]*template.Template),
	}

	templates := []string{
		"portal.html",
		"default.html",
		"result.html",
	}

	for _, tmpl := range templates {
		t, err := template.ParseFiles(filepath.Join("pages", tmpl))
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", tmpl, err)
		}
		BASIC_SERVICE.templates[tmpl] = t
	}
}

func (a *AuthServer) AuthChap(username []byte, chapid byte, chappwd, chapcha []byte, userip net.IP, usermac net.HardwareAddr) (err error, to uint32) {
	if info, ok := a.authing_user[userip.String()]; ok {
		if bytes.Equal(username, info.Name) && i.TestChapPwd(chapid, info.Pwd, chapcha, chappwd) {
			to = info.Timeout
			info.Mac = usermac
			return
		}
	} else {
		err = fmt.Errorf("radius auth - no such user %s", userip.String())
	}
	return
}

func (a *AuthServer) AuthMac(mac net.HardwareAddr, userip net.IP) (err error, to uint32) {
	err = fmt.Errorf("unsupported mac auth on %s", userip.String())
	to = 0
	return
}

func (a *AuthServer) AuthPap(username, userpwd []byte, userip net.IP) (err error, to uint32) {
	if info, ok := a.authing_user[userip.String()]; ok {
		if bytes.Equal(info.Pwd, userpwd) {
			to = info.Timeout
		}
	} else {
		err = fmt.Errorf("radius auth - no such user %s", userip.String())
	}
	return
}

func (a *AuthServer) HandlePortal(w http.ResponseWriter, r *http.Request) {
	handleResponse(w, Response{
		Code: http.StatusOK,
		Data: struct {
			Title   string
			Message string
			NasIP   string
			UserIP  string
			Timeout string
		}{
			Title:   "WiFi认证门户",
			Message: "欢迎使用网络服务，请登录以继续访问互联网",
			NasIP:   r.URL.Query().Get("nasip"),
			UserIP:  r.URL.Query().Get("userip"),
			Timeout: r.URL.Query().Get("timeout"),
		},
	}, "portal.html")
}

func (a *AuthServer) HandleLogin(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		handleResponse(w, Response{
			Code: http.StatusMethodNotAllowed,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登录失败",
				Message: "仅支持POST请求",
			},
		}, "result.html")
		return
	}

	if !strings.Contains(r.Header.Get("Referer"), "/portal") {
		handleResponse(w, Response{
			Code: http.StatusForbidden,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登录失败",
				Message: "请从Portal页面进行登录",
			},
		}, "result.html")
		return
	}

	if !config.IsValidClient(r.RemoteAddr) {
		handleResponse(w, Response{
			Code: http.StatusForbidden,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登录失败",
				Message: "该IP不在配置可允许的用户中",
			},
		}, "result.html")
		return
	}

	nasip_str := r.FormValue("nasip")
	if *config.NasIp != "" {
		nasip_str = *config.NasIp
	}
	userip_str := r.FormValue("userip")
	username := []byte(r.FormValue("username"))
	userpwd := []byte(r.FormValue("userpwd"))

	var userip net.IP
	if *config.UseRemoteIpAsUserIp {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		userip = net.ParseIP(ip)
	} else {
		userip = net.ParseIP(userip_str)
		if userip == nil {
			handleResponse(w, Response{
				Code: http.StatusBadRequest,
				Data: struct {
					Title   string
					Message string
				}{
					Title:   "登录失败",
					Message: "无效的用户IP地址",
				},
			}, "result.html")
			return
		}
	}

	nasip := net.ParseIP(nasip_str)
	if nasip == nil {
		handleResponse(w, Response{
			Code: http.StatusBadRequest,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登录失败",
				Message: "NAS IP配置错误",
			},
		}, "result.html")
		return
	}

	log.Printf("got a login request from %s on nas %s\n", userip, nasip)

	var full_username []byte
	if len(username) == 0 {
		handleResponse(w, Response{
			Code: http.StatusBadRequest,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登录失败",
				Message: "用户名不能为空",
			},
		}, "result.html")
		return
	}

	if *config.HuaweiDomain != "" {
		full_username = []byte(string(username) + "@" + *config.HuaweiDomain)
	} else {
		full_username = username
	}

	a.authing_user[userip.String()] = &AuthInfo{
		Name:    username,
		Pwd:     userpwd,
		Mac:     []byte{},
		Timeout: 3600,
	}

	if err := Auth(userip, nasip, full_username, userpwd); err != nil {
		log.Printf("Authentication failed: username %s in nas %s, err %v", full_username, nasip, err)
		handleResponse(w, Response{
			Code: http.StatusUnauthorized,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登录失败",
				Message: "用户名或密码错误",
			},
		}, "result.html")
		return
	}

	handleResponse(w, Response{
		Code: http.StatusOK,
		Data: struct {
			Title   string
			Message string
		}{
			Title:   "登录成功",
			Message: "您可以点击窗口右上角完成",
		},
	}, "result.html")
}

func (a *AuthServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	nas := r.FormValue("nasip")
	userip_str := r.FormValue("userip")

	userip := net.ParseIP(userip_str)
	if userip == nil {
		handleResponse(w, Response{
			Code: http.StatusBadRequest,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登出失败",
				Message: fmt.Sprintf("无效的用户IP地址: %s", userip_str),
			},
		}, "result.html")
		return
	}

	basip := net.ParseIP(nas)
	if basip == nil {
		handleResponse(w, Response{
			Code: http.StatusBadRequest,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登出失败",
				Message: fmt.Sprintf("无效的NAS IP地址: %s", nas),
			},
		}, "result.html")
		return
	}

	if _, err := Logout(userip, *config.HuaweiSecret, basip); err != nil {
		log.Printf("Logout failed: userip %s in nas %s, err %v", userip, basip, err)
		handleResponse(w, Response{
			Code: http.StatusInternalServerError,
			Data: struct {
				Title   string
				Message string
			}{
				Title:   "登出失败",
				Message: "登出请求失败，请稍后再试",
			},
		}, "result.html")
		return
	}

	handleResponse(w, Response{
		Code: http.StatusOK,
		Data: struct {
			Title   string
			Message string
		}{
			Title:   "登出成功",
			Message: "您已成功退出网络",
		},
	}, "result.html")
}

func (a *AuthServer) HandleRoot(w http.ResponseWriter, r *http.Request) {
	handleResponse(w, Response{
		Code: http.StatusOK,
		Data: struct {
			Title      string
			Message    string
			RemoteAddr string
			Path       string
			Timestamp  string
		}{
			Title:      "访问受限",
			Message:    "抱歉，您无权访问此页面。请通过正确的认证流程访问网络。",
			RemoteAddr: r.RemoteAddr,
			Path:       r.URL.Path,
			Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
		},
	}, "default.html")
}

func (a *AuthServer) AcctStart(username []byte, userip net.IP, nasip net.IP, usermac net.HardwareAddr, sessionid string) error {
	return nil
}

func (a *AuthServer) AcctStop(username []byte, userip net.IP, nasip net.IP, usermac net.HardwareAddr, sessionid string) error {
	return nil
}
