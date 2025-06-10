package server

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
	"github.com/spf13/viper"

	"syler/internal/logger"
	"syler/internal/sms"
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

type Authenticator struct {
	authUserInfo map[string]*AuthInfo
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

var AuthHandler = new(Authenticator)

func InitAuthenticator() {
	log := logger.GetLogger()

	AuthHandler = &Authenticator{
		authUserInfo: make(map[string]*AuthInfo),
		log:          log,
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         viper.GetString("redis.addr"),
		Password:     viper.GetString("redis.password"),
		DB:           viper.GetInt("redis.db"),
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MaxRetries:   3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
			"addr":  viper.GetString("redis.addr"),
		}).Fatal("Failed to connect to Redis")
	} else {
		AuthHandler.redisClient = rdb
		log.WithFields(logrus.Fields{
			"addr": viper.GetString("redis.addr"),
		}).Info("Redis connection initialized successfully")
	}

	if viper.GetString("sms.provider") != "" {
		smsConfig := sms.SMSConfig{
			Provider:     sms.Provider(viper.GetString("sms.provider")),
			AccessKey:    viper.GetString("sms.access_key"),
			SecretKey:    viper.GetString("sms.secret_key"),
			SignName:     viper.GetString("sms.sign_name"),
			TemplateCode: viper.GetString("sms.template_code"),
			Region:       viper.GetString("sms.region"),
			SDKAppID:     viper.GetString("sms.sdk_app_id"),
		}

		smsProvider, err := sms.NewSMSProvider(smsConfig)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error":    err,
				"provider": viper.GetString("sms.provider"),
			}).Fatal("Failed to initialize SMS provider")
		} else {
			AuthHandler.smsProvider = smsProvider
			log.WithFields(logrus.Fields{
				"provider": viper.GetString("sms.provider"),
			}).Info("SMS provider initialized successfully")
		}
	}
}

func (a *Authenticator) HandleLogin(w http.ResponseWriter, r *http.Request) {
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

	userip_str := r.FormValue("userip")
	userip := net.ParseIP(userip_str)
	if userip == nil {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: "无效的用户IP地址",
		})
		return
	}

	nasip_str := r.FormValue("nasip")
	nasip := net.ParseIP(nasip_str)
	if nasip == nil {
		handleResponse(w, http.StatusBadRequest, Response{
			Message: "NAS IP配置错误",
		})
		return
	}

	usermac_str := r.FormValue("usermac")
	username := []byte(r.FormValue("username"))
	userpwd := []byte(r.FormValue("userpwd"))

	log := logger.WithRequest(r).WithFields(logrus.Fields{
		"user_ip": userip,
		"nas_ip":  nasip,
	})
	log.Info("Received login request")

	if len(username) == 0 {
		log.Warn("Empty username provided")
		handleResponse(w, http.StatusBadRequest, Response{
			Message: "用户名不能为空",
		})
		return
	}

	a.authUserInfo[string(username)] = &AuthInfo{
		Name:  username,
		Pwd:   userpwd,
		Mac:   []byte{},
		IP:    userip,
		NasIP: nasip,
	}

	if err := Auth(userip, nasip, username, userpwd); err != nil {
		log.WithFields(logrus.Fields{
			"username": string(username),
			"error":    err,
		}).Error("Authentication failed")
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
			log.WithFields(logrus.Fields{
				"error": err,
				"mac":   usermac_str,
			}).Error("Failed to save MAC to Redis")
			handleResponse(w, http.StatusInternalServerError, Response{
				Message: "系统错误，请稍后重试",
			})
			return
		}
	} else {
		log.WithFields(logrus.Fields{
			"username": string(username),
		}).Info("No MAC address provided")
	}

	log.WithFields(logrus.Fields{
		"username": string(username),
	}).Info("User logged in successfully")

	handleResponse(w, http.StatusOK, Response{
		Message: "登录成功",
		Data: map[string]interface{}{
			"username": string(username),
			"userip":   userip.String(),
			"timeout":  "7天",
		},
	})
}

func (a *Authenticator) HandleLogout(w http.ResponseWriter, r *http.Request) {
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

	log := logger.WithRequest(r).WithFields(logrus.Fields{
		"user_ip": userip,
		"nas_ip":  nasip,
	})
	log.Info("Received logout request")

	if _, err := Logout(userip, nasip); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Logout failed")
		handleResponse(w, http.StatusConflict, Response{
			Message: "登出请求失败，请稍后再试",
		})
		return
	}

	log.Info("User logged out successfully")

	handleResponse(w, http.StatusOK, Response{
		Message: "登出成功",
	})
}

func (a *Authenticator) HandleRoot(w http.ResponseWriter, r *http.Request) {
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

func (a *Authenticator) HandleSendCode(w http.ResponseWriter, r *http.Request) {

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

	log := logger.WithRequest(r).WithFields(logrus.Fields{
		"phone": req.Phone,
	})

	if err := a.smsProvider.SendCode(req.Phone, code); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to send SMS")
		handleResponse(w, http.StatusInternalServerError, Response{
			Message: "发送验证码失败，请稍后重试",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	key := SMSCodePrefix + req.Phone
	if err := a.redisClient.SetEx(ctx, key, code, SMSCodeExpire).Err(); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to save code to Redis")
		handleResponse(w, http.StatusInternalServerError, Response{
			Message: "系统错误，请稍后重试",
		})
		return
	}

	log.Info("Successfully sent SMS code")

	handleResponse(w, http.StatusOK, Response{
		Message: "验证码已发送",
		Data: map[string]interface{}{
			"expire_seconds": int(SMSCodeExpire.Seconds()),
		},
	})
}
