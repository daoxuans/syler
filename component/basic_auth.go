package component

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"daoxuans/syler/config"
	"daoxuans/syler/logger"
	"daoxuans/syler/sms"
)

const (
	SMSCodePrefix     = "user:"         // Redis key prefix for SMS codes
	SMSCodeExpire     = 5 * time.Minute // Code expiration time
	MacSessionPfrefix = "mac:"          // Redis key prefix for MAC addresses
	MacSessionExpire  = 7 * 24 * time.Hour
)

type AuthInfo struct {
	Name  []byte
	Pwd   []byte
	Mac   net.HardwareAddr
	IP    net.IP
	NasIP net.IP
}

type AuthServer struct {
	authing_user map[string]*AuthInfo
	smsProvider  sms.SMSProvider
	redisClient  *redis.Client
	log          *logrus.Logger
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
	log := logger.GetLogger()

	BASIC_SERVICE = &AuthServer{
		authing_user: make(map[string]*AuthInfo),
		log:          log,
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         *config.RedisAddr,
		Password:     *config.RedisPassword,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MaxRetries:   3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Warning: Failed to connect to Redis: %v", err)
	} else {
		BASIC_SERVICE.redisClient = rdb
		log.Printf("Redis connection initialized successfully at %s", *config.RedisAddr)
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
	usermac_str := r.FormValue("usermac")
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

	a.log.Printf("got a login request from %s on nas %s\n", userip, nasip)

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

	a.authing_user[string(username)] = &AuthInfo{
		Name:  username,
		Pwd:   userpwd,
		Mac:   []byte{},
		IP:    userip,
		NasIP: nasip,
	}

	if err := Auth(userip, nasip, full_username, userpwd); err != nil {
		a.log.Printf("Authentication failed: username %s in nas %s, err %v", full_username, nasip, err)
		handleResponse(w, http.StatusUnauthorized, Response{
			Message: "用户名或密码错误",
		})
		return
	}

	if usermac_str != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		formatmac := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(usermac_str, ":", ""), "-", ""))
		key := MacSessionPfrefix + formatmac
		if err := a.redisClient.SetEx(ctx, key, 1, MacSessionExpire).Err(); err != nil {
			a.log.Printf("Failed to save mac to Redis: %v", err)
			handleResponse(w, http.StatusInternalServerError, Response{
				Message: "系统错误，请稍后重试",
			})
			return
		}
	} else {
		a.log.Printf("No MAC address provided for user %s on nas %s", username, nasip)
	}

	a.log.Printf("User %s logged in successfully from %s", username, nasip)

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

	a.log.Printf("got a logout request from %s on nas %s\n", userip, nasip)

	if _, err := Logout(userip, nasip); err != nil {
		a.log.Printf("Logout failed: userip %s on nas %s, err %v", userip, nasip, err)
		handleResponse(w, http.StatusConflict, Response{
			Message: "登出请求失败，请稍后再试",
		})
		return
	}

	a.log.Printf("User %s logged out successfully from NAS %s", userip, nasip)

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

func (a *AuthServer) HandleSendCode(w http.ResponseWriter, r *http.Request) {

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

	if !strings.Contains(r.Header.Get("Referer"), "/portal") {
		handleResponse(w, http.StatusForbidden, Response{
			Message: "请从Portal页面获取验证码",
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

	if !validatePhone(req.Phone) {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: "无效的手机号格式",
		})
		return
	}

	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	if err := a.smsProvider.SendCode(req.Phone, code); err != nil {
		a.log.Printf("Failed to send SMS to %s: provider=%s, error=%v",
			req.Phone,
			*config.SMSProvider,
			err,
		)
		handleResponse(w, http.StatusInternalServerError, Response{
			Message: "发送验证码失败，请稍后重试",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	key := SMSCodePrefix + req.Phone
	if err := a.redisClient.SetEx(ctx, key, code, SMSCodeExpire).Err(); err != nil {
		a.log.Printf("Failed to save code to Redis: %v", err)
		handleResponse(w, http.StatusInternalServerError, Response{
			Message: "系统错误，请稍后重试",
		})
		return
	}

	a.log.Printf("Successfully sent SMS code to %s", req.Phone)

	handleResponse(w, http.StatusOK, Response{
		Message: "验证码已发送",
		Data: map[string]interface{}{
			"expire_seconds": int(SMSCodeExpire.Seconds()),
		},
	})
}
