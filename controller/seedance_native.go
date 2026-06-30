package controller

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/doubao"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

type seedanceNativeCreateRequest struct {
	Model                 string               `json:"model"`
	Content               []doubao.ContentItem `json:"content"`
	CallbackURL           string               `json:"callback_url,omitempty"`
	ReturnLastFrame       *bool                `json:"return_last_frame,omitempty"`
	ServiceTier           string               `json:"service_tier,omitempty"`
	ExecutionExpiresAfter *int                 `json:"execution_expires_after,omitempty"`
	GenerateAudio         *bool                `json:"generate_audio,omitempty"`
	Priority              *int                 `json:"priority,omitempty"`
	Resolution            string               `json:"resolution,omitempty"`
	Ratio                 string               `json:"ratio,omitempty"`
	Duration              *int                 `json:"duration,omitempty"`
	Frames                *int                 `json:"frames,omitempty"`
	Seed                  *int                 `json:"seed,omitempty"`
	CameraFixed           *bool                `json:"camera_fixed,omitempty"`
	Watermark             *bool                `json:"watermark,omitempty"`
}

type seedanceNativeTaskResponse struct {
	ID                    string                 `json:"id"`
	Model                 string                 `json:"model"`
	Status                string                 `json:"status"`
	Error                 *seedanceNativeError   `json:"error"`
	CreatedAt             int64                  `json:"created_at"`
	UpdatedAt             int64                  `json:"updated_at"`
	Content               seedanceNativeContent  `json:"content"`
	Seed                  int                    `json:"seed,omitempty"`
	Resolution            string                 `json:"resolution,omitempty"`
	Duration              int                    `json:"duration,omitempty"`
	Ratio                 string                 `json:"ratio,omitempty"`
	FramesPerSecond       int                    `json:"framespersecond,omitempty"`
	ServiceTier           string                 `json:"service_tier,omitempty"`
	Usage                 seedanceNativeUsage    `json:"usage,omitempty"`
	Request               map[string]interface{} `json:"request,omitempty"`
	ExecutionExpiresAfter int                    `json:"execution_expires_after,omitempty"`
}

type seedanceNativeContent struct {
	VideoURL     string `json:"video_url"`
	LastFrameURL string `json:"last_frame_url,omitempty"`
}

type seedanceNativeUsage struct {
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type seedanceNativeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type seedanceNativeAPIError struct {
	status  int
	code    string
	message string
	errType string
}

func (e *seedanceNativeAPIError) Error() string {
	return e.message
}

type seedanceCanonicalTaskData struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Content struct {
		VideoURL     string `json:"video_url"`
		LastFrameURL string `json:"last_frame_url"`
	} `json:"content"`
	Seed                  int                    `json:"seed"`
	Resolution            string                 `json:"resolution"`
	Duration              int                    `json:"duration"`
	Ratio                 string                 `json:"ratio"`
	FramesPerSecond       int                    `json:"framespersecond"`
	ServiceTier           string                 `json:"service_tier"`
	Usage                 seedanceNativeUsage    `json:"usage"`
	Request               map[string]interface{} `json:"request"`
	ExecutionExpiresAfter int                    `json:"execution_expires_after"`
	Error                 seedanceNativeError    `json:"error"`
	CreatedAt             int64                  `json:"created_at"`
	UpdatedAt             int64                  `json:"updated_at"`
}

// prepareSeedanceNativeCreateRequest 将 BytePlus native create 请求转换为现有 OpenAI Video task 请求。
// 转换后复用 RelayTask 提交、计费和落库链路；native-only 字段保存在 metadata，供 task adaptor 和 canonical data 使用。
func prepareSeedanceNativeCreateRequest(c *gin.Context) error {
	var nativeReq seedanceNativeCreateRequest
	if err := common.UnmarshalBodyReusable(c, &nativeReq); err != nil {
		return newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.InvalidJSON", err.Error(), "BadRequest")
	}
	taskReq, err := convertSeedanceNativeCreateRequest(nativeReq)
	if err != nil {
		return err
	}
	body, err := common.Marshal(taskReq)
	if err != nil {
		return err
	}
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		return err
	}
	c.Set(common.KeyBodyStorage, storage)
	c.Request.Body = io.NopCloser(storage)
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("seedance_native_response", true)
	return nil
}

// SeedanceNativeCreate 适配 BytePlus / ModelArk native create 入口。
// 只做协议转换与 native response 标记，实际提交、预扣、结算和落库仍复用 RelayTask。
func SeedanceNativeCreate(c *gin.Context) {
	if err := prepareSeedanceNativeCreateRequest(c); err != nil {
		respondSeedanceNativeError(c, err)
		return
	}
	if !isSeedanceNativeChannelType(common.GetContextKeyInt(c, constant.ContextKeyChannelType)) {
		c.JSON(http.StatusNotFound, seedanceNativeErrorBody("InvalidEndpointOrModel.NotFound", "model is not available for Seedance native endpoint", "NotFound"))
		return
	}
	RelayTask(c)
}

// SeedanceNativeGet 按 public task id 查询当前 API Key 用户的 Seedance native 可渲染任务。
func SeedanceNativeGet(c *gin.Context) {
	userID := c.GetInt("id")
	taskID := c.Param("id")
	task, exists, err := model.GetByTaskId(userID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, seedanceNativeErrorBody("InternalServiceError", err.Error(), "InternalServerError"))
		return
	}
	if !exists || !isSeedanceNativeRenderableTask(task) {
		c.JSON(http.StatusNotFound, seedanceNativeErrorBody("ResourceNotFound.Task", "task not found", "NotFound"))
		return
	}

	resp, err := buildSeedanceNativeTaskResponse(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, seedanceNativeErrorBody("InternalServiceError", err.Error(), "InternalServerError"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// SeedanceNativeList 列出当前 API Key 用户最近 7 天内的 Seedance native 可渲染任务。
func SeedanceNativeList(c *gin.Context) {
	pageNum, pageSize, query, err := buildSeedanceNativeListQuery(c, seedanceNativeListNow())
	if err != nil {
		respondSeedanceNativeError(c, err)
		return
	}

	userID := c.GetInt("id")
	tasks := getAllSeedanceNativeListTasks(userID, query)
	items := make([]seedanceNativeTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		if !matchesSeedanceNativeModel(task, query.Model) {
			continue
		}
		item, buildErr := buildSeedanceNativeTaskResponse(task)
		if buildErr != nil {
			continue
		}
		items = append(items, item)
	}

	total := len(items)
	start := (pageNum - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	c.JSON(http.StatusOK, gin.H{
		"items":     items[start:end],
		"page_num":  pageNum,
		"page_size": pageSize,
		"total":     total,
	})
}

func buildSeedanceNativeTaskResponse(task *model.Task) (seedanceNativeTaskResponse, error) {
	var data seedanceCanonicalTaskData
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &data); err != nil {
			return seedanceNativeTaskResponse{}, err
		}
	}

	resp := seedanceNativeTaskResponse{
		ID:        task.TaskID,
		Model:     firstNonEmpty(task.Properties.OriginModelName, data.Model),
		Status:    mapSeedanceNativeStatus(task.Status, data.Status),
		CreatedAt: firstNonZeroInt64(data.CreatedAt, task.CreatedAt, task.SubmitTime),
		UpdatedAt: firstNonZeroInt64(data.UpdatedAt, task.UpdatedAt, task.FinishTime),
		Content: seedanceNativeContent{
			VideoURL:     data.Content.VideoURL,
			LastFrameURL: data.Content.LastFrameURL,
		},
		Seed:                  data.Seed,
		Resolution:            data.Resolution,
		Duration:              data.Duration,
		Ratio:                 data.Ratio,
		FramesPerSecond:       data.FramesPerSecond,
		ServiceTier:           data.ServiceTier,
		Usage:                 data.Usage,
		Request:               data.Request,
		ExecutionExpiresAfter: data.ExecutionExpiresAfter,
	}
	if resp.ServiceTier == "" {
		resp.ServiceTier = "default"
	}
	if task.Status == model.TaskStatusFailure || data.Status == "failed" {
		resp.Error = &seedanceNativeError{
			Code:    data.Error.Code,
			Message: firstNonEmpty(data.Error.Message, task.FailReason),
		}
	}
	return resp, nil
}

func isSeedanceNativeRenderableTask(task *model.Task) bool {
	if task == nil {
		return false
	}
	channelType, err := strconv.Atoi(string(task.Platform))
	if err != nil {
		return false
	}
	return isSeedanceNativeChannelType(channelType)
}

func seedanceNativePlatforms() []constant.TaskPlatform {
	return []constant.TaskPlatform{
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
	}
}

func isSeedanceNativeChannelType(channelType int) bool {
	return channelType == constant.ChannelTypeDoubaoVideo || channelType == constant.ChannelTypeVolcEngine
}

func matchesSeedanceNativeModel(task *model.Task, modelFilter string) bool {
	if modelFilter == "" {
		return true
	}
	modelName := task.Properties.OriginModelName
	if modelName == "" && len(task.Data) > 0 {
		var data struct {
			Model string `json:"model"`
		}
		if err := common.Unmarshal(task.Data, &data); err == nil {
			modelName = data.Model
		}
	}
	return modelName == modelFilter
}

func getAllSeedanceNativeListTasks(userID int, query model.SyncTaskQueryParams) []*model.Task {
	const batchSize = 500
	tasks := make([]*model.Task, 0)
	for offset := 0; ; offset += batchSize {
		batch := model.TaskGetAllUserTask(userID, offset, batchSize, query)
		tasks = append(tasks, batch...)
		if len(batch) < batchSize {
			return tasks
		}
	}
}

func buildSeedanceNativeListQuery(c *gin.Context, now int64) (int, int, model.SyncTaskQueryParams, error) {
	pageNum, err := parseSeedanceNativePageParam(c.Query("page_num"), "page_num", 1)
	if err != nil {
		return 0, 0, model.SyncTaskQueryParams{}, err
	}
	pageSize, err := parseSeedanceNativePageParam(c.Query("page_size"), "page_size", 10)
	if err != nil {
		return 0, 0, model.SyncTaskQueryParams{}, err
	}

	status, err := mapSeedanceNativeStatusFilter(c.Query("filter.status"))
	if err != nil {
		return 0, 0, model.SyncTaskQueryParams{}, err
	}
	serviceTier := c.Query("filter.service_tier")
	if serviceTier != "" && serviceTier != "default" {
		return 0, 0, model.SyncTaskQueryParams{}, newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.Unsupported", "filter.service_tier is not configurable", "BadRequest")
	}
	query := model.SyncTaskQueryParams{
		Action:         constant.TaskActionGenerate,
		Status:         status,
		TaskIDs:        c.QueryArray("filter.task_ids"),
		Model:          c.Query("filter.model"),
		Platforms:      seedanceNativePlatforms(),
		StartTimestamp: now - 7*24*3600,
		EndTimestamp:   now,
	}
	return pageNum, pageSize, query, nil
}

func convertSeedanceNativeCreateRequest(req seedanceNativeCreateRequest) (relaycommon.TaskSubmitReq, error) {
	prompt := ""
	for _, item := range req.Content {
		if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
			prompt = item.Text
			break
		}
	}
	if strings.TrimSpace(req.Model) == "" {
		return relaycommon.TaskSubmitReq{}, newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.InvalidValue", "model is required", "BadRequest")
	}
	if strings.TrimSpace(prompt) == "" {
		return relaycommon.TaskSubmitReq{}, newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.InvalidValue", "content text is required", "BadRequest")
	}
	if strings.TrimSpace(req.CallbackURL) != "" {
		return relaycommon.TaskSubmitReq{}, newSeedanceNativeAPIError(http.StatusBadRequest, "OperationDenied.CallbackNotSupported", "callback_url is not supported", "BadRequest")
	}
	if req.ServiceTier != "" && req.ServiceTier != "default" {
		return relaycommon.TaskSubmitReq{}, newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.Unsupported", "service_tier is not configurable", "BadRequest")
	}
	if req.Frames != nil {
		return relaycommon.TaskSubmitReq{}, newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.Unsupported", "frames is not supported", "BadRequest")
	}
	if req.Seed != nil {
		return relaycommon.TaskSubmitReq{}, newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.Unsupported", "seed is not supported", "BadRequest")
	}
	if req.CameraFixed != nil {
		return relaycommon.TaskSubmitReq{}, newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.Unsupported", "camera_fixed is not supported", "BadRequest")
	}

	metadata := map[string]interface{}{
		"content": req.Content,
	}
	putOptionalString(metadata, "service_tier", req.ServiceTier)
	putOptionalString(metadata, "resolution", req.Resolution)
	putOptionalString(metadata, "ratio", req.Ratio)
	putOptionalBool(metadata, "return_last_frame", req.ReturnLastFrame)
	putOptionalBool(metadata, "generate_audio", req.GenerateAudio)
	putOptionalBool(metadata, "camera_fixed", req.CameraFixed)
	putOptionalBool(metadata, "watermark", req.Watermark)
	putOptionalInt(metadata, "execution_expires_after", req.ExecutionExpiresAfter)
	putOptionalInt(metadata, "priority", req.Priority)
	putOptionalInt(metadata, "duration", req.Duration)
	putOptionalInt(metadata, "frames", req.Frames)
	putOptionalInt(metadata, "seed", req.Seed)

	taskReq := relaycommon.TaskSubmitReq{
		Model:    req.Model,
		Prompt:   prompt,
		Metadata: metadata,
	}
	if req.Duration != nil {
		taskReq.Seconds = strconv.Itoa(*req.Duration)
		taskReq.Duration = *req.Duration
	}
	return taskReq, nil
}

func mapSeedanceNativeStatus(status model.TaskStatus, dataStatus string) string {
	switch status {
	case model.TaskStatusSubmitted, model.TaskStatusQueued, model.TaskStatusNotStart:
		return "queued"
	case model.TaskStatusInProgress:
		return "running"
	case model.TaskStatusSuccess:
		return "succeeded"
	case model.TaskStatusFailure:
		return "failed"
	default:
		if dataStatus != "" {
			return dataStatus
		}
		return "running"
	}
}

func mapSeedanceNativeStatusFilter(status string) (string, error) {
	switch status {
	case "":
		return "", nil
	case "queued":
		return string(model.TaskStatusQueued), nil
	case "running":
		return string(model.TaskStatusInProgress), nil
	case "succeeded":
		return string(model.TaskStatusSuccess), nil
	case "failed":
		return string(model.TaskStatusFailure), nil
	case "cancelled":
		return "__seedance_native_no_match__", nil
	default:
		return "", newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.InvalidValue", "filter.status is invalid", "BadRequest")
	}
}

func parseSeedanceNativePageParam(raw string, name string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 500 {
		return 0, newSeedanceNativeAPIError(http.StatusBadRequest, "InvalidParameter.InvalidValue", fmt.Sprintf("%s must be an integer in [1,500]", name), "BadRequest")
	}
	return value, nil
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func seedanceNativeListNow() int64 {
	return time.Now().Unix()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func seedanceNativeErrorBody(code string, message string, errType string) gin.H {
	return gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
			"type":    errType,
		},
	}
}

func respondSeedanceNativeError(c *gin.Context, err error) {
	if apiErr, ok := err.(*seedanceNativeAPIError); ok {
		c.JSON(apiErr.status, seedanceNativeErrorBody(apiErr.code, apiErr.message, apiErr.errType))
		return
	}
	c.JSON(http.StatusInternalServerError, seedanceNativeErrorBody("InternalServiceError", err.Error(), "InternalServerError"))
}

func newSeedanceNativeAPIError(status int, code string, message string, errType string) *seedanceNativeAPIError {
	return &seedanceNativeAPIError{
		status:  status,
		code:    code,
		message: message,
		errType: errType,
	}
}

func putOptionalString(metadata map[string]interface{}, key string, value string) {
	if value != "" {
		metadata[key] = value
	}
}

func putOptionalBool(metadata map[string]interface{}, key string, value *bool) {
	if value != nil {
		metadata[key] = *value
	}
}

func putOptionalInt(metadata map[string]interface{}, key string, value *int) {
	if value != nil {
		metadata[key] = *value
	}
}
