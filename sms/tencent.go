package sms

import (
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

type TencentSMS struct {
	client     *sms.Client
	sdkAppId   string
	signName   string
	templateId string
}

func NewTencentSMS(secretId, secretKey, region, sdkAppId, signName, templateId string) (*TencentSMS, error) {
	credential := common.NewCredential(secretId, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"

	client, err := sms.NewClient(credential, region, cpf)
	if err != nil {
		return nil, err
	}

	return &TencentSMS{
		client:     client,
		sdkAppId:   sdkAppId,
		signName:   signName,
		templateId: templateId,
	}, nil
}

func (s *TencentSMS) SendCode(phone, code string) error {
	request := sms.NewSendSmsRequest()
	request.SmsSdkAppId = common.StringPtr(s.sdkAppId)
	request.SignName = common.StringPtr(s.signName)
	request.TemplateId = common.StringPtr(s.templateId)
	request.PhoneNumberSet = common.StringPtrs([]string{"+86" + phone})
	request.TemplateParamSet = common.StringPtrs([]string{code})

	response, err := s.client.SendSms(request)
	if err != nil {
		return err
	}

	for _, status := range response.Response.SendStatusSet {
		if *status.Code != "Ok" {
			return fmt.Errorf("send SMS failed: %s", *status.Message)
		}
	}

	return nil
}
