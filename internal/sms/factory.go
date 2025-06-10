package sms

import "fmt"

type Provider string

const (
	ProviderAliyun  Provider = "aliyun"
	ProviderTencent Provider = "tencent"
)

// SMSConfig 短信配置
type SMSConfig struct {
	Provider     Provider
	AccessKey    string
	SecretKey    string
	SignName     string
	TemplateCode string
	Region       string // 腾讯云特有
	SDKAppID     string // 腾讯云特有
}

// NewSMSProvider 创建短信服务提供商实例
func NewSMSProvider(config SMSConfig) (SMSProvider, error) {
	switch config.Provider {
	case ProviderAliyun:
		return NewAliyunSMS(
			config.AccessKey,
			config.SecretKey,
			config.SignName,
			config.TemplateCode,
		)
	case ProviderTencent:
		return NewTencentSMS(
			config.AccessKey,
			config.SecretKey,
			config.Region,
			config.SDKAppID,
			config.SignName,
			config.TemplateCode,
		)
	default:
		return nil, fmt.Errorf("unsupported SMS provider: %s", config.Provider)
	}
}
