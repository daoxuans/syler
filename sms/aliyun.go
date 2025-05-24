package sms

import (
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

type AliyunSMS struct {
	client       *dysmsapi.Client
	signName     string
	templateCode string
}

func NewAliyunSMS(accessKey, secretKey, signName, templateCode string) (*AliyunSMS, error) {
	client, err := dysmsapi.NewClientWithAccessKey(
		"cn-hangzhou",
		accessKey,
		secretKey,
	)
	if err != nil {
		return nil, err
	}

	return &AliyunSMS{
		client:       client,
		signName:     signName,
		templateCode: templateCode,
	}, nil
}

func (s *AliyunSMS) SendCode(phone, code string) error {
	request := dysmsapi.CreateSendSmsRequest()
	request.Scheme = "https"
	request.PhoneNumbers = phone
	request.SignName = s.signName
	request.TemplateCode = s.templateCode
	request.TemplateParam = fmt.Sprintf("{\"code\":\"%s\"}", code)

	response, err := s.client.SendSms(request)
	if err != nil {
		return err
	}

	if response.Code != "OK" {
		return fmt.Errorf("send SMS failed: %s", response.Message)
	}

	return nil
}
