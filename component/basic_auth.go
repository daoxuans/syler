package component

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"daoxuans/syler/config"
	"daoxuans/syler/i"
)

type AuthInfo struct {
	Name    []byte
	Pwd     []byte
	Mac     net.HardwareAddr
	Timeout uint32
}

type AuthServer struct {
	authing_user map[string]*AuthInfo
	templates    map[string]*template.Template // 添加模板缓存
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func jsonResponse(w http.ResponseWriter, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

var BASIC_SERVICE = new(AuthServer)

func InitBasic() {
	BASIC_SERVICE = &AuthServer{
		authing_user: make(map[string]*AuthInfo),
		templates:    make(map[string]*template.Template),
	}

	// 初始化时加载所有模板
	templates := []string{"portal.html", "root.html"}
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
	tmpl, ok := a.templates["portal.html"]
	if !ok {
		log.Printf("Portal template not found in cache")
		jsonResponse(w, http.StatusInternalServerError, "Internal Server Error", nil)
		return
	}

	data := struct {
		NasIP   string
		UserIP  string
		Timeout string
	}{
		NasIP:   r.URL.Query().Get("nasip"),
		UserIP:  r.URL.Query().Get("userip"),
		Timeout: r.URL.Query().Get("timeout"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execution failed: %v", err)
	}
}

func (a *AuthServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// 1. 验证请求方法
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, "仅支持POST请求", nil)
		return
	}

	// 2. 验证来源
	referer := r.Header.Get("Referer")
	if referer == "" || !strings.Contains(r.Header.Get("Referer"), "/portal") {
		jsonResponse(w, http.StatusForbidden, "请从Portal页面进行登录", nil)
		return
	}

	// 3. 验证客户端
	if !config.IsValidClient(r.RemoteAddr) {
		jsonResponse(w, http.StatusForbidden, "该IP不在配置可允许的用户中", nil)
		return
	}

	// 4. 解析请求参数
	timeout := r.FormValue("timeout")
	nas := r.FormValue("nasip")
	if *config.NasIp != "" {
		nas = *config.NasIp
	}
	userip_str := r.FormValue("userip")
	username := []byte(r.FormValue("username"))
	userpwd := []byte(r.FormValue("userpwd"))

	// 5. 处理超时时间
	to, err := strconv.ParseUint(timeout, 10, 32)
	if err != nil || (to == 0 && *config.DefaultTimeout != 0) {
		to = *config.DefaultTimeout
	}

	// 6. 处理用户IP
	var userip net.IP
	if *config.UseRemoteIpAsUserIp {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		userip = net.ParseIP(ip)
	} else {
		userip = net.ParseIP(userip_str)
		if userip == nil {
			jsonResponse(w, http.StatusBadRequest, "无效的用户IP地址", nil)
			return
		}
	}

	// 7. 处理NAS IP
	basip := net.ParseIP(nas)
	if basip == nil {
		jsonResponse(w, http.StatusBadRequest, "NAS IP配置错误", nil)
		return
	}

	// 8. 处理用户认证
	log.Printf("got a login request from %s on nas %s\n", userip, basip)

	var full_username []byte
	if len(username) == 0 {
		if !*config.RandomUser {
			jsonResponse(w, http.StatusBadRequest, "username required", nil)
			return
		}
		full_username, userpwd = a.RandomUser(userip, basip, *config.HuaweiDomain, uint32(to))
	} else {
		full_username = []byte(string(username) + "@" + *config.HuaweiDomain)
		a.authing_user[userip.String()] = &AuthInfo{
			Name:    username,
			Pwd:     userpwd,
			Mac:     []byte{},
			Timeout: uint32(to),
		}
	}

	// 9. 执行认证
	if err = Auth(userip, basip, uint32(to), full_username, userpwd); err != nil {
		log.Printf("Authentication failed: %v", err)
		jsonResponse(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// 10. 返回成功响应
	jsonResponse(w, http.StatusOK, "login successful", map[string]string{"mac": a.authing_user[userip.String()].Mac.String()})
}

// 处理Logout请求
func (a *AuthServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	nas := r.FormValue("nasip")
	userip_str := r.FormValue("userip")

	userip := net.ParseIP(userip_str)
	if userip == nil {
		jsonResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid user IP: %s", userip_str), nil)
		return
	}

	basip := net.ParseIP(nas)
	if basip == nil {
		jsonResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid NAS IP: %s", nas), nil)
		return
	}

	if _, err := Logout(userip, *config.HuaweiSecret, basip); err != nil {
		jsonResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	jsonResponse(w, http.StatusOK, "logout successful", nil)
}

func (a *AuthServer) HandleRoot(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := a.templates["root.html"]
	if !ok {
		log.Printf("Root template not found in cache")
		jsonResponse(w, http.StatusInternalServerError, "Internal Server Error", nil)
		return
	}

	data := struct {
		RemoteAddr string
		Path       string
		Timestamp  string
	}{
		RemoteAddr: r.RemoteAddr,
		Path:       r.URL.Path,
		Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execution failed: %v", err)
	}
}

func (a *AuthServer) RandomUser(userip, nasip net.IP, domain string, timeout uint32) ([]byte, []byte) {
	hash := md5.New()
	hash.Write(userip)
	hash.Write(nasip)
	bts := hash.Sum(nil)
	username := []byte(userip.String())
	app := []byte("@" + domain)
	if len(username)+len(app) > 32 {
		username = username[:32-len(app)]
	}
	fname := append(username, app...)
	userpwd := bts
	a.authing_user[userip.String()] = &AuthInfo{username, userpwd, []byte{}, timeout}
	return fname, userpwd
}

func (a *AuthServer) AcctStart(username []byte, userip net.IP, nasip net.IP, usermac net.HardwareAddr, sessionid string) error {
	return nil
}

func (a *AuthServer) AcctStop(username []byte, userip net.IP, nasip net.IP, usermac net.HardwareAddr, sessionid string) error {
	callBackOffline(*config.CallBackUrl, userip, nasip)
	return nil
}

func (a *AuthServer) NotifyLogout(userip, nasip net.IP) error {
	callBackOffline(*config.CallBackUrl, userip, nasip)
	return nil
}
