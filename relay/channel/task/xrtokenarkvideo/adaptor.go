package xrtokenarkvideo

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	ID string `json:"id"`
}

type responseTask struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Content struct {
		VideoURL     string `json:"video_url,omitempty"`
		LastFrameURL string `json:"last_frame_url,omitempty"`
	} `json:"content,omitempty"`
	VideoURL              string                  `json:"video_url"`
	LastFrameURL          string                  `json:"last_frame_url,omitempty"`
	Duration              any                     `json:"duration,omitempty"`
	CreatedAt             common.FlexibleUnixTime `json:"created_at"`
	UpdatedAt             common.FlexibleUnixTime `json:"updated_at"`
	Seed                  int                     `json:"seed,omitempty"`
	Resolution            string                  `json:"resolution,omitempty"`
	Ratio                 string                  `json:"ratio,omitempty"`
	FramesPerSecond       int                     `json:"framespersecond,omitempty"`
	ServiceTier           string                  `json:"service_tier,omitempty"`
	ExecutionExpiresAfter int                     `json:"execution_expires_after,omitempty"`
	GenerateAudio         bool                    `json:"generate_audio,omitempty"`
	Draft                 bool                    `json:"draft,omitempty"`
	Priority              int                     `json:"priority,omitempty"`
	Usage                 struct {
		CompletionTokens int `json:"completion_tokens,omitempty"`
		TotalTokens      int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (r responseTask) resultVideoURL() string {
	if r.Content.VideoURL != "" {
		return r.Content.VideoURL
	}
	return r.VideoURL
}

// TaskAdaptor 适配 XRToken ARK 视频任务协议。
// 它复用现有 OpenAI Video 入口和 Doubao Seedance 请求体，只覆盖上游路径和顶层结果字段解析。
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

// ValidateRequestAndSetAction 解析任务请求并固定为视频生成动作。
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL 构造 XRToken 上游创建任务 URL。
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v1/contents/generations/tasks", a.baseURL), nil
}

// BuildRequestHeader 设置 XRToken 上游请求头。
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

// DoResponse 解析 XRToken 创建响应，内部返回上游 task id，对用户返回公开 task id。
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var xResp responseTask
	if err := common.Unmarshal(responseBody, &xResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}
	if xResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	if c.GetBool("seedance_native_response") {
		taskData = responseBody
		if canonicalData, buildErr := taskdoubao.BuildNativeCreateTaskData(c, info, xResp.ID); buildErr == nil {
			taskData = canonicalData
		}
		c.JSON(http.StatusOK, gin.H{"id": info.PublicTaskID})
		return xResp.ID, taskData, nil
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	if xResp.CreatedAt > 0 {
		ov.CreatedAt = xResp.CreatedAt.Unix()
	}
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return xResp.ID, responseBody, nil
}

// FetchTask 从 XRToken 视频生成查询端点查询上游任务状态。
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/videos/generations/%s", baseUrl, url.PathEscape(taskID))
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

// GetModelList 返回当前 XRToken ARK 视频渠道支持的公开模型列表。
func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

// GetChannelName 返回当前 task adaptor 的渠道标识。
func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// ParseTaskResult 将 XRToken 查询响应映射为统一任务状态。
// 查询接口可能返回顶层 video_url，也可能返回 Seedance native content.video_url，两者都兼容。
func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

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
	case "succeeded", "completed":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		taskResult.Url = resTask.resultVideoURL()
		taskResult.CompletionTokens = resTask.Usage.CompletionTokens
		taskResult.TotalTokens = resTask.Usage.TotalTokens
	case "failed":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = resTask.Error.Message
	default:
		// 未知上游状态继续轮询，避免把未识别状态误判为失败并触发退款。
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

// ConvertToOpenAIVideo 将保存的 XRToken 原始任务数据转换为 OpenAI Video 查询外壳。
func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var xResp responseTask
	if err := common.Unmarshal(originTask.Data, &xResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal xrtoken ark video task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.TaskID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	resultURL := xResp.resultVideoURL()
	if resultURL == "" {
		// 轮询成功但上游未返回 video_url 时，任务结果会落到代理 URL。
		resultURL = originTask.GetResultURL()
	}
	openAIVideo.SetMetadata("url", resultURL)
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt
	if xResp.CreatedAt > 0 {
		openAIVideo.CreatedAt = xResp.CreatedAt.Unix()
	}
	if xResp.UpdatedAt > 0 {
		openAIVideo.CompletedAt = xResp.UpdatedAt.Unix()
	}
	openAIVideo.Seconds = durationToString(xResp.Duration)
	openAIVideo.Model = originTask.Properties.OriginModelName

	if xResp.Status == "failed" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: xResp.Error.Message,
			Code:    xResp.Error.Code,
		}
	}

	return common.Marshal(openAIVideo)
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
