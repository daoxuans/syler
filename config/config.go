package config

import (
	"net"
	"strings"

	toml "github.com/extrame/go-toml-config"
)

var (
	HttpPort            = toml.Int("http.port", 8080)
	HttpWhiteList       = toml.String("http.white_list", "")
	NasIp               = toml.String("http.nas_ip", "")
	UseRemoteIpAsUserIp = toml.Bool("http.remote_ip_as_user_ip", false)
	SMSProvider         = toml.String("sms.provider", "")
	SMSAccessKey        = toml.String("sms.access_key", "")
	SMSSecretKey        = toml.String("sms.secret_key", "")
	SMSSignName         = toml.String("sms.sign_name", "")
	SMSTemplateCode     = toml.String("sms.template_code", "")
	SMSRegion           = toml.String("sms.region", "ap-guangzhou")
	SMSSDKAppID         = toml.String("sms.sdk_app_id", "")
	RedisAddr           = toml.String("redis.addr", "localhost:6379")
	RedisPassword       = toml.String("redis.password", "")
	HuaweiPort          = toml.Int("huawei.port", 50100)
	HuaweiVersion       = toml.Int("huawei.version", 1)
	HuaweiSecret        = toml.String("huawei.secret", "testing123")
	HuaweiNasPort       = toml.Int("huawei.nas_port", 2000)
	HuaweiDomain        = toml.String("huawei.domain", "huawei.com")
	LogFile             = toml.String("basic.logfile", "")
)

func IsValid() bool {
	return true
}

func IsValidClient(addr string) bool {
	if *HttpWhiteList == "" {
		return true
	}
	if ip, _, err := net.SplitHostPort(addr); err == nil {
		if strings.Contains(*HttpWhiteList, ip) {
			return true
		}
	}
	return false
}
