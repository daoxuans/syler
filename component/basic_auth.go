package component

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"daoxuans/syler/config"
	"daoxuans/syler/i"
	"daoxuans/syler/sms"
)

type AuthInfo struct {
	Name    []byte
	Pwd     []byte
	Mac     net.HardwareAddr
	Timeout uint32 //用户会话超时时间，单位秒
}

type AuthServer struct {
	authing_user map[string]*AuthInfo
	smsProvider  sms.SMSProvider
}

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func handleResponse(w http.ResponseWriter, httpStatus int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(resp)
}

func validatePhone(phone string) bool {
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

var BASIC_SERVICE = new(AuthServer)

func InitBasic() {
	BASIC_SERVICE = &AuthServer{
		authing_user: make(map[string]*AuthInfo),
	}

	if config.SMSProvider != nil && *config.SMSProvider != "" {
		smsConfig := sms.SMSConfig{
			Provider:     sms.Provider(*config.SMSProvider),
			AccessKey:    *config.SMSAccessKey,
			SecretKey:    *config.SMSSecretKey,
			SignName:     *config.SMSSignName,
			TemplateCode: *config.SMSTemplateCode,
			Region:       *config.SMSRegion,
			SDKAppID:     *config.SMSSDKAppID,
		}

		smsProvider, err := sms.NewSMSProvider(smsConfig)
		if err != nil {
			log.Fatalf("Warning: Failed to initialize SMS provider: %v", err)
		} else {
			BASIC_SERVICE.smsProvider = smsProvider
			log.Printf("SMS provider %s initialized successfully", *config.SMSProvider)
		}
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

func (a *AuthServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handleResponse(w, http.StatusMethodNotAllowed, Response{
			Message: "仅支持POST请求",
		})
		return
	}

	if !strings.Contains(r.Header.Get("Referer"), "/portal") {
		handleResponse(w, http.StatusForbidden, Response{
			Message: "请从Portal页面进行登录",
		})
		return
	}

	if !config.IsValidClient(r.RemoteAddr) {
		handleResponse(w, http.StatusForbidden, Response{
			Message: "该IP不在配置可允许的用户中",
		})
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
			handleResponse(w, http.StatusBadRequest, Response{
				Message: "无效的用户IP地址",
			})
			return
		}
	}

	nasip := net.ParseIP(nasip_str)
	if nasip == nil {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: "NAS IP配置错误",
		})
		return
	}

	log.Printf("got a login request from %s on nas %s\n", userip, nasip)

	var full_username []byte
	if len(username) == 0 {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: "用户名不能为空",
		})
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
		Timeout: 604800,
	}

	if err := Auth(userip, nasip, full_username, userpwd); err != nil {
		log.Printf("Authentication failed: username %s in nas %s, err %v", full_username, nasip, err)
		handleResponse(w, http.StatusUnauthorized, Response{
			Message: "用户名或密码错误",
		})
		return
	}

	handleResponse(w, http.StatusOK, Response{
		Message: "登录成功",
		Data: map[string]interface{}{
			"username": string(username),
			"userip":   userip.String(),
			"timeout":  "7天",
		},
	})
}

func (a *AuthServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	nas := r.FormValue("nasip")
	userip_str := r.FormValue("userip")

	userip := net.ParseIP(userip_str)
	if userip == nil {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: fmt.Sprintf("无效的用户IP地址: %s", userip_str),
		})
		return
	}

	nasip := net.ParseIP(nas)
	if nasip == nil {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: fmt.Sprintf("无效的NAS IP地址: %s", nas),
		})
		return
	}

	log.Printf("got a logout request from %s on nas %s\n", userip, nasip)

	if _, err := Logout(userip, nasip); err != nil {
		log.Printf("Logout failed: userip %s on nas %s, err %v", userip, nasip, err)
		handleResponse(w, http.StatusConflict, Response{
			Message: "登出请求失败，请稍后再试",
		})
		return
	}

	handleResponse(w, http.StatusOK, Response{
		Message: "登出成功",
	})
}

func (a *AuthServer) HandleRoot(w http.ResponseWriter, r *http.Request) {
	handleResponse(w, http.StatusOK, Response{
		Message: "抱歉，您无权访问此页面。请通过正确的认证流程访问网络。",
		Data: struct {
			RemoteAddr string
			Path       string
			Timestamp  string
		}{
			RemoteAddr: r.RemoteAddr,
			Path:       r.URL.Path,
			Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}

func (a *AuthServer) AcctStart(username []byte, userip net.IP, nasip net.IP, usermac net.HardwareAddr, sessionid string) error {
	return nil
}

func (a *AuthServer) AcctStop(username []byte, userip net.IP, nasip net.IP, usermac net.HardwareAddr, sessionid string) error {
	return nil
}

func (a *AuthServer) HandleSendCode(w http.ResponseWriter, r *http.Request) {
	// 检查是否启用了短信服务
	if a.smsProvider == nil {
		handleResponse(w, http.StatusServiceUnavailable, Response{
			Message: "短信服务未启用",
		})
		return
	}

	if r.Method != http.MethodPost {
		handleResponse(w, http.StatusMethodNotAllowed, Response{
			Message: "仅支持POST请求",
		})
		return
	}

	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: "无效的请求参数",
		})
		return
	}

	if validatePhone(req.Phone) {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: "无效的手机号格式",
		})
		return
	}

	// 生成6位随机验证码
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// 发送验证码
	err := a.smsProvider.SendCode(req.Phone, code)
	if err != nil {
		log.Printf("Failed to send SMS to %s: %v", req.Phone, err)
		handleResponse(w, http.StatusInternalServerError, Response{
			Message: "发送验证码失败，请稍后重试",
		})
		return
	}

	// 保存验证码和发送时间
	// 这里可以使用 Redis 或数据库来存储验证码和过期时间
	// 例如：saveCodeToDB(req.Phone, code, time.Now().Add(5*time.Minute))

	handleResponse(w, http.StatusOK, Response{
		Message: "验证码已发送",
		Data: map[string]interface{}{
			"expire_seconds": 300, // 5分钟有效期
		},
	})
}
