package sms

// SMSProvider 短信服务商接口
type SMSProvider interface {
	SendCode(phone, code string) error
}
