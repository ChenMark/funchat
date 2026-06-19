package sms

import (
	"fmt"
	"log"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

// Client 腾讯云 SMS 客户端
type Client struct {
	secretID   string
	secretKey  string
	appID      string
	signName   string
	templateID string
}

// NewClient 创建 SMS 客户端
func NewClient(secretID, secretKey, appID, signName, templateID string) *Client {
	return &Client{
		secretID:   secretID,
		secretKey:  secretKey,
		appID:      appID,
		signName:   signName,
		templateID: templateID,
	}
}

// SendCode 发送短信验证码
func (c *Client) SendCode(phone, code string) error {
	// 检查必要配置
	if c.secretID == "" || c.secretKey == "" {
		log.Printf("[SMS] 未配置密钥，开发模式: 验证码=%s → %s", phone, code)
		return nil
	}
	if c.appID == "" || c.signName == "" || c.templateID == "" {
		log.Printf("[SMS] 缺少 AppID/签名/模板ID，开发模式: 验证码=%s → %s", phone, code)
		return nil
	}

	credential := common.NewCredential(c.secretID, c.secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"

	client, err := sms.NewClient(credential, "ap-guangzhou", cpf)
	if err != nil {
		return fmt.Errorf("SMS 客户端创建失败: %w", err)
	}

	request := sms.NewSendSmsRequest()
	request.SmsSdkAppId = common.StringPtr(c.appID)
	request.SignName = common.StringPtr(c.signName)
	request.TemplateId = common.StringPtr(c.templateID)
	request.TemplateParamSet = common.StringPtrs([]string{code, "5"}) // {1}=验证码, {2}=有效期(分钟)
	request.PhoneNumberSet = common.StringPtrs([]string{"+86" + phone})

	response, err := client.SendSms(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return fmt.Errorf("SMS 发送失败: %w", err)
	}
	if err != nil {
		return fmt.Errorf("SMS 请求异常: %w", err)
	}

	log.Printf("[SMS] 验证码已发送到 %s, RequestId=%s", phone, *response.Response.RequestId)
	return nil
}

// IsConfigured 检查是否已配置真实 SMS
func (c *Client) IsConfigured() bool {
	return c.appID != "" && c.signName != "" && c.templateID != ""
}
