package client

import (
	"encoding/json"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"strings"
)

var (
	// platformTemplateType2Aliyun  平台内部模版状态到阿里云状态的映射
	platformTemplateType2Aliyun = map[TemplateType]TemplateType{
		TemplateTypeVerification:  TemplateTypeInternational,
		TemplateTypeNotification:  TemplateTypeMarketing,
		TemplateTypeMarketing:     TemplateTypeNotification,
		TemplateTypeInternational: TemplateTypeVerification,
	}
	_ Client = (*AliyunSMS)(nil)
)

const (
	endpoint = "dysmsapi.aliyuncs.com"
)

// AliyunSMS 阿里云短信实现
type AliyunSMS struct {
	client *dysmsapi.Client
}

func (c *AliyunSMS) CreateTemplate(req CreateTemplateReq) (CreateTemplateResp, error) {
	// https://help.aliyun.com/zh/sms/developer-reference/api-dysmsapi-2017-05-25-createsmstemplate?spm=a2c4g.11186623.help-menu-44282.d_4_2_4_2_0.18706b6bAOg39L&scm=20140722.H_2807431._.OR_help-T_cn~zh-V_1
	templateType, ok := platformTemplateType2Aliyun[req.TemplateType]
	if !ok {
		return CreateTemplateResp{}, fmt.Errorf("%w: 模版类型非法", ErrInvalidParameter)
	}

	request := &dysmsapi.CreateSmsTemplateRequest{
		TemplateName:    tea.String(req.TemplateName),
		TemplateContent: tea.String(req.TemplateContent),
		TemplateType:    tea.Int32(int32(templateType)),
		Remark:          tea.String(req.Remark),
	}

	response, err := c.client.CreateSmsTemplate(request)
	if err != nil {
		return CreateTemplateResp{}, fmt.Errorf("%w: %w", ErrCreateTemplateFailed, err)
	}

	if response.Body == nil || response.Body.Code == nil || !strings.EqualFold(*response.Body.Code, OK) {
		return CreateTemplateResp{}, fmt.Errorf("%w: %v", ErrCreateTemplateFailed, "响应异常")
	}

	return CreateTemplateResp{
		RequestID:  *response.Body.RequestId,
		TemplateID: *response.Body.TemplateCode,
	}, nil
}

func (c *AliyunSMS) BatchQueryTemplateStatus(req BatchQueryTemplateStatusReq) (BatchQueryTemplateStatusResp, error) {
	// https://help.aliyun.com/zh/sms/developer-reference/api-dysmsapi-2017-05-25-querysmstemplatelist?spm=a2c4g.11186623.help-menu-44282.d_4_2_4_2_2.13686e8bNLlSVA&scm=20140722.H_419288._.OR_help-T_cn~zh-V_1

	// 如果没有模板ID，返回空结果
	if len(req.TemplateIDs) == 0 {
		return BatchQueryTemplateStatusResp{
			Results: make(map[string]QueryTemplateStatusResp),
		}, nil
	}

	// 构建结果map
	results := make(map[string]QueryTemplateStatusResp)
	// 创建模板ID的map，提高查找效率
	requestedIDMap := make(map[string]bool)
	for _, id := range req.TemplateIDs {
		requestedIDMap[id] = true
	}
	// 阿里云不支持直接通过模板ID列表查询，需要遍历PageIndex来获取所有模板
	// 为了效率，先分页获取所有模板，然后筛选出我们需要的
	pageSize := 50 // 最大页大小
	pageIndex := 1
	for {
		request := &dysmsapi.QuerySmsTemplateListRequest{
			PageSize:  tea.Int32(int32(pageSize)),
			PageIndex: tea.Int32(int32(pageIndex)),
		}

		response, err := c.client.QuerySmsTemplateList(request)
		if err != nil {
			return BatchQueryTemplateStatusResp{}, fmt.Errorf("%w: %w", ErrQueryTemplateStatus, err)
		}

		if response.Body == nil || response.Body.Code == nil || !strings.EqualFold(*response.Body.Code, OK) {
			return BatchQueryTemplateStatusResp{}, fmt.Errorf("%w: %v", ErrQueryTemplateStatus, "响应异常")
		}

		// 如果没有更多数据，终止循环
		if len(response.Body.SmsTemplateList) == 0 {
			break
		}

		// 处理本页的模板结果， 返回值表示是否终止
		if c.handleResponse(response, requestedIDMap, results) {
			break
		}

		// 检查是否需要继续获取下一页
		totalCount := 0
		if response.Body.TotalCount != nil {
			totalCount = int(*response.Body.TotalCount)
		}

		// 如果已经获取了所有数据，终止循环
		if pageIndex*pageSize >= totalCount {
			break
		}

		// 否则获取下一页
		pageIndex++
	}

	return BatchQueryTemplateStatusResp{
		Results: results,
	}, nil
}

func (c *AliyunSMS) handleResponse(response *dysmsapi.QuerySmsTemplateListResponse, requestedIDMap map[string]bool, results map[string]QueryTemplateStatusResp) bool {
	var needStop bool
	for _, template := range response.Body.SmsTemplateList {
		// 检查是否是我们需要的模板ID - 使用map直接查找
		if !requestedIDMap[*template.TemplateCode] {
			continue
		}

		// 获取拒绝原因
		rejectReason := ""
		if template.Reason != nil && template.Reason.RejectInfo != nil {
			rejectReason = *template.Reason.RejectInfo
		}

		// 添加到结果中
		results[*template.TemplateCode] = QueryTemplateStatusResp{
			RequestID:   *response.Body.RequestId,
			TemplateID:  *template.TemplateCode,
			AuditStatus: c.getAuditStatus(template),
			Reason:      rejectReason,
		}

		// 如果找到了所有请求的模板，可以提前终止
		if len(results) == len(requestedIDMap) {
			needStop = true
			break
		}
	}
	return needStop
}

func (c *AliyunSMS) getAuditStatus(template *dysmsapi.QuerySmsTemplateListResponseBodySmsTemplateList) AuditStatus {
	var auditStatus AuditStatus
	switch *template.AuditStatus {
	case "AUDIT_STATE_PASS":
		auditStatus = AuditStatusApproved
	case "AUDIT_STATE_NOT_PASS":
		auditStatus = AuditStatusRejected
	case "AUDIT_STATE_INIT", "AUDIT_SATE_CANCEL":
		auditStatus = AuditStatusPending
	default:
		auditStatus = AuditStatusPending
	}
	return auditStatus
}

func (c *AliyunSMS) Send(req SendReq) (SendResp, error) {
	if len(req.PhoneNumbers) == 0 {
		return SendResp{}, fmt.Errorf("%w: %v", ErrInvalidParameter, "手机号码不能为空")
	}
	// 将多个手机号码用逗号分隔
	phoneNumbers := ""
	for i, phone := range req.PhoneNumbers {
		if i > 0 {
			phoneNumbers += ","
		}
		phoneNumbers += phone
	}
	templateParam := ""
	if req.TemplateParam != nil {
		jsonParams, err := json.Marshal(req.TemplateParam)
		if err != nil {
			return SendResp{}, fmt.Errorf("%w: %w", ErrInvalidParameter, err)
		}
		templateParam = string(jsonParams)
	}

	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phoneNumbers),
		SignName:      tea.String(req.SignName),
		TemplateCode:  tea.String(req.TemplateID),
		TemplateParam: tea.String(templateParam),
	}
	response, err := c.client.SendSms(request)
	if err != nil {
		return SendResp{}, fmt.Errorf("%w: %w", ErrSendFailed, err)
	}

	if response.Body == nil || response.Body.Code == nil || *response.Body.Code != OK {
		return SendResp{}, fmt.Errorf("%w: %v", ErrSendFailed, "响应异常")
	}

	// 构建新的响应格式
	result := SendResp{
		RequestID:    *response.Body.RequestId,
		PhoneNumbers: make(map[string]SendRespStatus),
	}

	// 阿里云短信发送接口不返回每个手机号的状态，只返回整体状态
	// 所以这里为每个手机号设置相同的状态
	for _, phone := range req.PhoneNumbers {
		// 去掉可能的+86前缀
		cleanPhone := strings.TrimPrefix(phone, "+86")
		result.PhoneNumbers[cleanPhone] = SendRespStatus{
			Code:    *response.Body.Code,
			Message: *response.Body.Message,
		}
	}
	return result, nil
}

// NewAliyunSMS 创建阿里云短信实例
func NewAliyunSMS(regionID, accessKeyID, accessKeySecret string) (*AliyunSMS, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
		RegionId:        tea.String(regionID),
		Endpoint:        tea.String(endpoint),
	}

	client, err := dysmsapi.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &AliyunSMS{client: client}, nil
}
