package serviceinferencevideo

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskdoubao "github.com/QuantumNous/new-api/relay/channel/task/doubao"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type responsePayload struct {
	Task responseTask `json:"task"`
}

type responseTask struct {
	ID              string         `json:"id"`
	Model           string         `json:"model"`
	Status          string         `json:"status"`
	DurationSeconds any            `json:"duration_seconds,omitempty"`
	Outputs         []string       `json:"outputs,omitempty"`
	LastFrameURL    string         `json:"last_frame_url,omitempty"`
	Error           *responseError `json:"error,omitempty"`
	CreatedAt       string         `json:"created_at,omitempty"`
	CompletedAt     string         `json:"completed_at,omitempty"`
	Usage           responseUsage  `json:"usage,omitempty"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type responseUsage struct {
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// TaskAdaptor 适配 service-inference.ai 视频任务协议。
// 它复用现有 OpenAI Video 入口和 Doubao Seedance 请求体，只覆盖上游路径、响应包裹和 outputs[] 解析。
type TaskAdaptor struct {
	taskcommon.BaseBilling
	doubao taskdoubao.TaskAdaptor

	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
	a.doubao.Init(info)
}

// ValidateRequestAndSetAction 解析视频任务请求并固定为生成动作。
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL 构造 service-inference.ai 创建视频任务 URL。
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v1/video/generate", a.baseURL), nil
}

// BuildRequestHeader 设置 service-inference.ai 上游请求头。
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// EstimateBilling 复用 Doubao Seedance 视频输入计费估算，保持公开模型名费率语义一致。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	return a.doubao.EstimateBilling(c, info)
}

// BuildRequestBody 复用 Doubao Seedance 请求体转换，只由 model_mapping 决定上游模型名。
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	return a.doubao.BuildRequestBody(c, info)
}

// DoRequest 复用 task adaptor 通用请求发送逻辑。
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse 解析 service-inference.ai 创建响应，内部返回上游 task.id，对用户返回公开 task id。
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var sResp responsePayload
	if err := common.Unmarshal(responseBody, &sResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}
	if sResp.Task.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	if createdAt := parseProviderTime(sResp.Task.CreatedAt); createdAt > 0 {
		ov.CreatedAt = createdAt
	}
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return sResp.Task.ID, responseBody, nil
}

// FetchTask 从 service-inference.ai 单任务端点查询上游任务状态。
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/video/tasks/%s", baseUrl, taskID)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

// GetModelList 返回当前 service-inference.ai 视频渠道支持的公开模型列表。
func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

// GetChannelName 返回当前 task adaptor 的渠道标识。
func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// ParseTaskResult 将 service-inference.ai 查询响应映射为统一任务状态。
// 成功状态只取 outputs[0] 作为主结果 URL，完整 outputs[] 由轮询流程保存在 Task.Data。
func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resPayload := responsePayload{}
	if err := common.Unmarshal(respBody, &resPayload); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	resTask := resPayload.Task
	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	switch resTask.Status {
	case "pending", "queued":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = "10%"
	case "processing", "running", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	case "completed", "succeeded", "success":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		if len(resTask.Outputs) > 0 {
			taskResult.Url = resTask.Outputs[0]
		}
		taskResult.CompletionTokens = resTask.Usage.CompletionTokens
		taskResult.TotalTokens = resTask.Usage.TotalTokens
	case "failed", "error", "cancelled", "canceled":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = responseErrorMessage(resTask.Error)
	default:
		// 未知上游状态继续轮询，避免把未识别状态误判为失败并触发退款。
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

// ConvertToOpenAIVideo 将保存的 service-inference.ai 原始任务数据转换为 OpenAI Video 查询外壳。
func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var sPayload responsePayload
	if err := common.Unmarshal(originTask.Data, &sPayload); err != nil {
		return nil, errors.Wrap(err, "unmarshal service-inference video task data failed")
	}
	sResp := sPayload.Task

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.TaskID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	resultURL := firstOutputURL(sResp.Outputs)
	if resultURL == "" {
		// 轮询成功但上游未返回 outputs 时，任务结果会落到代理 URL 或历史兼容字段。
		resultURL = originTask.GetResultURL()
	}
	openAIVideo.SetMetadata("url", resultURL)
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt
	if createdAt := parseProviderTime(sResp.CreatedAt); createdAt > 0 {
		openAIVideo.CreatedAt = createdAt
	}
	if completedAt := parseProviderTime(sResp.CompletedAt); completedAt > 0 {
		openAIVideo.CompletedAt = completedAt
	}
	openAIVideo.Seconds = durationToString(sResp.DurationSeconds)
	openAIVideo.Model = originTask.Properties.OriginModelName

	if isFailureStatus(sResp.Status) {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: responseErrorMessage(sResp.Error),
			Code:    responseErrorCode(sResp.Error),
		}
	}

	return common.Marshal(openAIVideo)
}

func firstOutputURL(outputs []string) string {
	if len(outputs) == 0 {
		return ""
	}
	return outputs[0]
}

func responseErrorMessage(errResp *responseError) string {
	if errResp == nil {
		return ""
	}
	return errResp.Message
}

func responseErrorCode(errResp *responseError) string {
	if errResp == nil {
		return ""
	}
	return errResp.Code
}

func isFailureStatus(status string) bool {
	switch status {
	case "failed", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func parseProviderTime(value string) int64 {
	if value == "" {
		return 0
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t.Unix()
	}
	return 0
}

func durationToString(duration any) string {
	switch v := duration.(type) {
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	default:
		return ""
	}
}
