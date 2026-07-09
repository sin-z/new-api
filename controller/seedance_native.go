package controller

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type seedanceNativeContentItem struct {
	Type     string `json:"type,omitempty"`
	Text     string `json:"text,omitempty"`
	ImageURL any    `json:"image_url,omitempty"`
	VideoURL any    `json:"video_url,omitempty"`
	AudioURL any    `json:"audio_url,omitempty"`
	Role     string `json:"role,omitempty"`
}

type seedanceNativeRequest struct {
	Model                 string                      `json:"model"`
	Content               []seedanceNativeContentItem `json:"content"`
	CallbackURL           string                      `json:"callback_url,omitempty"`
	ReturnLastFrame       *bool                       `json:"return_last_frame,omitempty"`
	ServiceTier           string                      `json:"service_tier,omitempty"`
	ExecutionExpiresAfter *int                        `json:"execution_expires_after,omitempty"`
	GenerateAudio         *bool                       `json:"generate_audio,omitempty"`
	Resolution            string                      `json:"resolution,omitempty"`
	Ratio                 string                      `json:"ratio,omitempty"`
	Duration              *int                        `json:"duration,omitempty"`
	Frames                *int                        `json:"frames,omitempty"`
	Seed                  *int                        `json:"seed,omitempty"`
	CameraFixed           *bool                       `json:"camera_fixed,omitempty"`
	Watermark             *bool                       `json:"watermark,omitempty"`
	Priority              *int                        `json:"priority,omitempty"`
}

type seedanceNativeValidationError struct {
	code    string
	message string
}

type seedanceNativeTaskResponse struct {
	ID                    string                      `json:"id"`
	Model                 string                      `json:"model"`
	Status                string                      `json:"status"`
	Error                 *seedanceNativeTaskError    `json:"error"`
	CreatedAt             int64                       `json:"created_at"`
	UpdatedAt             int64                       `json:"updated_at"`
	Content               seedanceNativeTaskContent   `json:"content"`
	Seed                  int                         `json:"seed,omitempty"`
	Resolution            string                      `json:"resolution,omitempty"`
	Ratio                 string                      `json:"ratio,omitempty"`
	Duration              int                         `json:"duration,omitempty"`
	FramesPerSecond       int                         `json:"framespersecond,omitempty"`
	GenerateAudio         *bool                       `json:"generate_audio,omitempty"`
	Draft                 bool                        `json:"draft,omitempty"`
	Priority              int                         `json:"priority,omitempty"`
	ServiceTier           string                      `json:"service_tier,omitempty"`
	ExecutionExpiresAfter int                         `json:"execution_expires_after,omitempty"`
	Usage                 *seedanceNativeTaskUsage    `json:"usage,omitempty"`
	Request               *seedanceNativeTaskSnapshot `json:"-"`
}

type seedanceNativeTaskContent struct {
	VideoURL     string `json:"video_url"`
	LastFrameURL string `json:"last_frame_url,omitempty"`
}

type seedanceNativeTaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type seedanceNativeTaskUsage struct {
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type seedanceNativeTaskSnapshot struct {
	Resolution            string `json:"resolution"`
	Ratio                 string `json:"ratio"`
	Duration              int    `json:"duration"`
	GenerateAudio         *bool  `json:"generate_audio"`
	ExecutionExpiresAfter int    `json:"execution_expires_after"`
	Priority              int    `json:"priority"`
}

type seedanceCanonicalTaskData struct {
	ID                    string                     `json:"id"`
	Model                 string                     `json:"model"`
	Status                string                     `json:"status"`
	Content               seedanceNativeTaskContent  `json:"content"`
	Error                 seedanceNativeTaskError    `json:"error"`
	Request               seedanceNativeTaskSnapshot `json:"request"`
	Usage                 seedanceNativeTaskUsage    `json:"usage"`
	Seed                  int                        `json:"seed"`
	ServiceTier           string                     `json:"service_tier"`
	CreatedAt             common.FlexibleUnixTime    `json:"created_at"`
	UpdatedAt             common.FlexibleUnixTime    `json:"updated_at"`
	Resolution            string                     `json:"resolution"`
	Ratio                 string                     `json:"ratio"`
	Duration              int                        `json:"duration"`
	FramesPerSecond       int                        `json:"framespersecond,omitempty"`
	GenerateAudio         *bool                      `json:"generate_audio"`
	Draft                 bool                       `json:"draft,omitempty"`
	ExecutionExpiresAfter int                        `json:"execution_expires_after"`
	Priority              int                        `json:"priority"`
	VideoURL              string                     `json:"video_url"`
	LastFrameURL          string                     `json:"last_frame_url,omitempty"`
}

type seedanceNativeTaskListResponse struct {
	Items []*seedanceNativeTaskResponse `json:"items"`
	Total int64                         `json:"total"`
}

// SeedanceNativeTaskCreate 将 Seedance native create 请求在 handler 内转成 OpenAI Video task 请求。
// 该入口只做外部协议适配和 native 响应渲染；提交、计费、落库和重试仍复用现有 relay task 能力。
func SeedanceNativeTaskCreate(c *gin.Context) {
	if !seedanceNativeTokenAuth(c) {
		return
	}
	nativeReq, taskReq, ok := seedanceNativeBuildOpenAIRequest(c)
	if !ok {
		return
	}
	if !seedanceNativeDistribute(c, nativeReq.Model) {
		return
	}
	if !seedanceNativeRewriteRequest(c, taskReq) {
		return
	}
	seedanceNativeRelayTask(c)
}

func seedanceNativeBuildOpenAIRequest(c *gin.Context) (seedanceNativeRequest, relaycommon.TaskSubmitReq, bool) {
	var nativeReq seedanceNativeRequest
	if err := common.UnmarshalBodyReusable(c, &nativeReq); err != nil {
		RenderSeedanceNativeError(c, http.StatusBadRequest, "InvalidParameter.InvalidJSON", "invalid request body")
		return nativeReq, relaycommon.TaskSubmitReq{}, false
	}
	if err := validateSeedanceNativeCreateRequest(nativeReq); err != nil {
		RenderSeedanceNativeError(c, http.StatusBadRequest, err.code, err.message)
		return nativeReq, relaycommon.TaskSubmitReq{}, false
	}
	metadata, err := seedanceNativeMetadata(nativeReq)
	if err != nil {
		RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", "failed to build request metadata")
		return nativeReq, relaycommon.TaskSubmitReq{}, false
	}
	taskReq := relaycommon.TaskSubmitReq{
		Prompt:   firstSeedanceTextPrompt(nativeReq.Content),
		Model:    nativeReq.Model,
		Metadata: metadata,
	}
	return nativeReq, taskReq, true
}

func seedanceNativeRewriteRequest(c *gin.Context, taskReq relaycommon.TaskSubmitReq) bool {
	jsonData, err := common.Marshal(taskReq)
	if err != nil {
		RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", "failed to marshal request body")
		return false
	}
	common.CleanupBodyStorage(c)
	c.Set(common.KeyRequestBody, jsonData)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
	c.Request.ContentLength = int64(len(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.URL.Path = "/v1/video/generations"
	c.Set("relay_mode", relayconstant.RelayModeVideoSubmit)
	c.Set("seedance_native_response", true)
	return true
}

func validateSeedanceNativeCreateRequest(req seedanceNativeRequest) *seedanceNativeValidationError {
	if strings.TrimSpace(req.Model) == "" {
		return &seedanceNativeValidationError{code: "InvalidParameter.InvalidValue", message: "model is required"}
	}
	if len(req.Content) == 0 {
		return &seedanceNativeValidationError{code: "InvalidParameter.InvalidValue", message: "content is required"}
	}
	if strings.TrimSpace(firstSeedanceTextPrompt(req.Content)) == "" {
		return &seedanceNativeValidationError{code: "InvalidParameter.InvalidValue", message: "content must contain a text item"}
	}
	if strings.TrimSpace(req.CallbackURL) != "" {
		return &seedanceNativeValidationError{code: "OperationDenied.CallbackNotSupported", message: "callback_url is not supported in P0"}
	}
	if req.ServiceTier != "" && req.ServiceTier != "default" {
		return &seedanceNativeValidationError{code: "InvalidParameter.Unsupported", message: "service_tier is not configurable for Seedance 2.0"}
	}
	if req.Duration != nil && *req.Duration != -1 && (*req.Duration < 4 || *req.Duration > 15) {
		return &seedanceNativeValidationError{code: "InvalidParameter.InvalidValue", message: "duration must be -1 or between 4 and 15 for Seedance 2.0"}
	}
	if req.Frames != nil {
		return &seedanceNativeValidationError{code: "InvalidParameter.Unsupported", message: "frames is not supported for Seedance 2.0"}
	}
	if req.Seed != nil {
		return &seedanceNativeValidationError{code: "InvalidParameter.Unsupported", message: "seed is not supported for Seedance 2.0"}
	}
	if req.CameraFixed != nil {
		return &seedanceNativeValidationError{code: "InvalidParameter.Unsupported", message: "camera_fixed is not supported for Seedance 2.0"}
	}
	return nil
}

func firstSeedanceTextPrompt(content []seedanceNativeContentItem) string {
	for _, item := range content {
		if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
			return item.Text
		}
	}
	return ""
}

func seedanceNativeMetadata(req seedanceNativeRequest) (map[string]interface{}, error) {
	raw, err := common.Marshal(req)
	if err != nil {
		return nil, err
	}
	var metadata map[string]interface{}
	if err := common.Unmarshal(raw, &metadata); err != nil {
		return nil, err
	}
	if _, ok := metadata["service_tier"]; !ok {
		metadata["service_tier"] = "default"
	}
	return metadata, nil
}

func seedanceNativeRelayTask(c *gin.Context) {
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", err.Error())
		return
	}
	if taskErr := relay.ResolveOriginTask(c, relayInfo); taskErr != nil {
		renderSeedanceTaskError(c, taskErr)
		return
	}

	var result *relay.TaskSubmitResult
	var taskErr *dto.TaskError
	defer func() {
		if taskErr != nil && relayInfo.Billing != nil {
			relayInfo.Billing.Refund(c)
		}
	}()

	retryParam := &service.RetryParam{
		Ctx:         c,
		TokenGroup:  relayInfo.TokenGroup,
		ModelName:   relayInfo.OriginModelName,
		RequestPath: c.Request.URL.Path,
		Retry:       common.GetPointer(0),
	}
	for ; retryParam.GetRetry() <= common.RetryTimes; retryParam.IncreaseRetry() {
		var channel *model.Channel
		if lockedCh, ok := relayInfo.LockedChannel.(*model.Channel); ok && lockedCh != nil {
			channel = lockedCh
			if retryParam.GetRetry() > 0 {
				if setupErr := middleware.SetupContextForSelectedChannel(c, channel, relayInfo.OriginModelName); setupErr != nil {
					taskErr = service.TaskErrorWrapperLocal(setupErr.Err, "setup_locked_channel_failed", http.StatusInternalServerError)
					break
				}
			}
		} else {
			var channelErr *types.NewAPIError
			channel, channelErr = getChannel(c, relayInfo, retryParam)
			if channelErr != nil {
				logger.LogError(c, channelErr.Error())
				taskErr = service.TaskErrorWrapperLocal(channelErr.Err, "get_channel_failed", http.StatusInternalServerError)
				break
			}
		}

		addUsedChannel(c, channel.Id)
		bodyStorage, bodyErr := common.GetBodyStorage(c)
		if bodyErr != nil {
			if common.IsRequestBodyTooLargeError(bodyErr) || errors.Is(bodyErr, common.ErrRequestBodyTooLarge) {
				taskErr = service.TaskErrorWrapperLocal(bodyErr, "read_request_body_failed", http.StatusRequestEntityTooLarge)
			} else {
				taskErr = service.TaskErrorWrapperLocal(bodyErr, "read_request_body_failed", http.StatusBadRequest)
			}
			break
		}
		c.Request.Body = io.NopCloser(bodyStorage)

		result, taskErr = relay.RelayTaskSubmit(c, relayInfo)
		if taskErr == nil {
			break
		}
		if !taskErr.LocalError {
			processChannelError(c,
				*types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey,
					common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()),
				types.NewOpenAIError(taskErr.Error, types.ErrorCodeBadResponseStatusCode, taskErr.StatusCode))
		}
		if !shouldRetrySeedanceTaskRelay(c, taskErr, common.RetryTimes-retryParam.GetRetry()) {
			break
		}
	}

	if taskErr == nil {
		if settleErr := service.SettleBilling(c, relayInfo, result.Quota); settleErr != nil {
			common.SysError("settle task billing error: " + settleErr.Error())
		}
		service.LogTaskConsumption(c, relayInfo)

		task := model.InitTask(result.Platform, relayInfo)
		task.PrivateData.UpstreamTaskID = result.UpstreamTaskID
		task.PrivateData.BillingSource = relayInfo.BillingSource
		task.PrivateData.SubscriptionId = relayInfo.SubscriptionId
		task.PrivateData.TokenId = relayInfo.TokenId
		task.PrivateData.BillingContext = &model.TaskBillingContext{
			ModelPrice:      relayInfo.PriceData.ModelPrice,
			GroupRatio:      relayInfo.PriceData.GroupRatioInfo.GroupRatio,
			ModelRatio:      relayInfo.PriceData.ModelRatio,
			OtherRatios:     relayInfo.PriceData.OtherRatios(),
			OriginModelName: relayInfo.OriginModelName,
			PerCallBilling:  common.StringsContains(constant.TaskPricePatches, relayInfo.OriginModelName) || relayInfo.PriceData.UsePrice,
		}
		task.Quota = result.Quota
		task.Data = result.TaskData
		task.Action = relayInfo.Action
		if insertErr := task.Insert(); insertErr != nil {
			common.SysError("insert task error: " + insertErr.Error())
		}
		if channelID := common.GetContextKeyInt(c, constant.ContextKeyChannelId); channelID > 0 && c.Writer != nil && c.Writer.Status() < http.StatusBadRequest {
			service.RecordChannelAffinity(c, channelID)
		}
		return
	}
	renderSeedanceTaskError(c, taskErr)
}

func shouldRetrySeedanceTaskRelay(c *gin.Context, taskErr *dto.TaskError, retryTimes int) bool {
	if taskErr == nil {
		return false
	}
	if service.ShouldSkipRetryAfterChannelAffinityFailure(c) {
		return false
	}
	if retryTimes <= 0 {
		return false
	}
	if _, ok := c.Get("specific_channel_id"); ok {
		return false
	}
	if taskErr.StatusCode == http.StatusTooManyRequests || taskErr.StatusCode == 307 {
		return true
	}
	if taskErr.StatusCode/100 == 5 {
		return !operation_setting.IsAlwaysSkipRetryStatusCode(taskErr.StatusCode)
	}
	if taskErr.StatusCode == http.StatusBadRequest || taskErr.StatusCode == 408 {
		return false
	}
	if taskErr.LocalError {
		return false
	}
	return taskErr.StatusCode/100 != 2
}

func renderSeedanceTaskError(c *gin.Context, taskErr *dto.TaskError) {
	if taskErr == nil {
		RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", "internal task error")
		return
	}
	if taskErr.StatusCode == http.StatusTooManyRequests {
		taskErr.Message = "当前分组上游负载已饱和，请稍后再试"
	}
	RenderSeedanceNativeError(c, seedanceNativeStatusCode(taskErr.StatusCode), seedanceTaskErrorCode(taskErr), taskErr.Message)
}

func seedanceTaskErrorCode(taskErr *dto.TaskError) string {
	if taskErr == nil {
		return "InternalServiceError"
	}
	if taskErr.Code != "" {
		return taskErr.Code
	}
	switch taskErr.StatusCode {
	case http.StatusBadRequest:
		return "InvalidParameter.InvalidValue"
	case http.StatusForbidden, http.StatusUnauthorized:
		return "OperationDenied.ServiceNotOpen"
	case http.StatusNotFound:
		return "ResourceNotFound.Task"
	case http.StatusTooManyRequests:
		return "AccountRateLimitExceeded"
	default:
		return "InternalServiceError"
	}
}

func seedanceNativeStatusCode(status int) int {
	if status == http.StatusUnauthorized {
		return http.StatusForbidden
	}
	if status == http.StatusServiceUnavailable {
		return http.StatusNotFound
	}
	return status
}

func seedanceNativeTokenAuth(c *gin.Context) bool {
	if c.GetInt("id") != 0 {
		return true
	}
	key, parts := seedanceNativeTokenKey(c)
	token, err := model.ValidateUserToken(key)
	if token != nil && c.GetInt("id") == 0 {
		c.Set("id", token.UserId)
	}
	if err != nil {
		if errors.Is(err, model.ErrDatabase) {
			common.SysLog("SeedanceNative ValidateUserToken database error: " + err.Error())
			RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", common.TranslateMessage(c, i18n.MsgDatabaseError))
		} else {
			RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", common.TranslateMessage(c, i18n.MsgTokenInvalid))
		}
		return false
	}

	if !seedanceNativeCheckTokenIP(c, token) {
		return false
	}
	userCache, err := model.GetUserCache(token.UserId)
	if err != nil {
		common.SysLog(fmt.Sprintf("SeedanceNative GetUserCache error for user %d: %v", token.UserId, err))
		RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", common.TranslateMessage(c, i18n.MsgDatabaseError))
		return false
	}
	if userCache.Status != common.UserStatusEnabled {
		RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", common.TranslateMessage(c, i18n.MsgAuthUserBanned))
		return false
	}
	userCache.WriteContext(c)

	userGroup := userCache.Group
	tokenGroup := token.Group
	if tokenGroup != "" {
		if _, ok := service.GetUserUsableGroups(userGroup)[tokenGroup]; !ok {
			RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", fmt.Sprintf("无权访问 %s 分组", tokenGroup))
			return false
		}
		if !ratio_setting.ContainsGroupRatio(tokenGroup) && tokenGroup != "auto" {
			RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", fmt.Sprintf("分组 %s 已被弃用", tokenGroup))
			return false
		}
		userGroup = tokenGroup
	}
	common.SetContextKey(c, constant.ContextKeyUsingGroup, userGroup)
	if !seedanceNativeSetupContextForToken(c, token, parts...) {
		return false
	}
	return true
}

func seedanceNativeSetupContextForToken(c *gin.Context, token *model.Token, parts ...string) bool {
	if token == nil {
		RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", "token is invalid")
		return false
	}
	c.Set("id", token.UserId)
	c.Set("token_id", token.Id)
	c.Set("token_key", token.Key)
	c.Set("token_name", token.Name)
	c.Set("token_unlimited_quota", token.UnlimitedQuota)
	if !token.UnlimitedQuota {
		c.Set("token_quota", token.RemainQuota)
	}
	if token.ModelLimitsEnabled {
		c.Set("token_model_limit_enabled", true)
		c.Set("token_model_limit", token.GetModelLimitsMap())
	} else {
		c.Set("token_model_limit_enabled", false)
	}
	common.SetContextKey(c, constant.ContextKeyTokenGroup, token.Group)
	common.SetContextKey(c, constant.ContextKeyTokenCrossGroupRetry, token.CrossGroupRetry)
	if len(parts) > 1 {
		if !model.IsAdmin(token.UserId) {
			RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", "普通用户不支持指定渠道")
			return false
		}
		c.Set("specific_channel_id", parts[1])
	}
	return true
}

func seedanceNativeTokenKey(c *gin.Context) (string, []string) {
	key := c.Request.Header.Get("Authorization")
	if strings.HasPrefix(key, "Bearer ") || strings.HasPrefix(key, "bearer ") {
		key = strings.TrimSpace(key[7:])
	}
	if key == "" || key == "midjourney-proxy" {
		key = c.Request.Header.Get("mj-api-secret")
		if strings.HasPrefix(key, "Bearer ") || strings.HasPrefix(key, "bearer ") {
			key = strings.TrimSpace(key[7:])
		}
		key = strings.TrimPrefix(key, "sk-")
		parts := strings.Split(key, "-")
		return parts[0], parts
	}
	key = strings.TrimPrefix(key, "sk-")
	parts := strings.Split(key, "-")
	return parts[0], parts
}

func seedanceNativeCheckTokenIP(c *gin.Context, token *model.Token) bool {
	allowIps := token.GetIpLimits()
	if len(allowIps) == 0 {
		return true
	}
	clientIP := c.ClientIP()
	logger.LogDebug(c, "Token has IP restrictions, checking client IP %s", clientIP)
	ip := net.ParseIP(clientIP)
	if ip == nil {
		RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", "无法解析客户端 IP 地址")
		return false
	}
	if !common.IsIpInCIDRList(ip, allowIps) {
		RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", "您的 IP 不在令牌允许访问的列表中")
		return false
	}
	return true
}

func seedanceNativeDistribute(c *gin.Context, modelName string) bool {
	if strings.TrimSpace(modelName) == "" {
		RenderSeedanceNativeError(c, http.StatusBadRequest, "InvalidParameter.InvalidValue", "model is required")
		return false
	}
	if common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled) {
		s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
		if !ok {
			RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", "token has no model access")
			return false
		}
		tokenModelLimit, ok := s.(map[string]bool)
		if !ok {
			tokenModelLimit = map[string]bool{}
		}
		matchName := ratio_setting.FormatMatchingModelName(modelName)
		if _, ok := tokenModelLimit[matchName]; !ok {
			RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", fmt.Sprintf("token has no access to model %s", modelName))
			return false
		}
	}

	channel, ok := seedanceNativeSelectChannel(c, modelName)
	if !ok {
		return false
	}
	common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
	if setupErr := middleware.SetupContextForSelectedChannel(c, channel, modelName); setupErr != nil {
		RenderSeedanceNativeError(c, seedanceNativeStatusCode(setupErr.StatusCode), "InternalServiceError", setupErr.Error())
		return false
	}
	return true
}

func seedanceNativeSelectChannel(c *gin.Context, modelName string) (*model.Channel, bool) {
	if channelId, ok := common.GetContextKey(c, constant.ContextKeyTokenSpecificChannelId); ok {
		id, err := strconv.Atoi(channelId.(string))
		if err != nil {
			RenderSeedanceNativeError(c, http.StatusBadRequest, "InvalidParameter.InvalidValue", "invalid channel id")
			return nil, false
		}
		channel, err := model.GetChannelById(id, true)
		if err != nil || channel == nil {
			RenderSeedanceNativeError(c, http.StatusBadRequest, "InvalidParameter.InvalidValue", "invalid channel id")
			return nil, false
		}
		if channel.Status != common.ChannelStatusEnabled {
			RenderSeedanceNativeError(c, http.StatusForbidden, "OperationDenied.ServiceNotOpen", "channel is disabled")
			return nil, false
		}
		return channel, true
	}

	usingGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	channel, selectGroup, err := service.CacheGetRandomSatisfiedChannel(&service.RetryParam{
		Ctx:         c,
		ModelName:   modelName,
		TokenGroup:  usingGroup,
		RequestPath: "/v1/video/generations",
		Retry:       common.GetPointer(0),
	})
	if err != nil {
		showGroup := usingGroup
		if usingGroup == "auto" {
			showGroup = fmt.Sprintf("auto(%s)", selectGroup)
		}
		RenderSeedanceNativeError(c, http.StatusNotFound, "InvalidEndpointOrModel.NotFound", fmt.Sprintf("分组 %s 下模型 %s 无可用渠道: %s", showGroup, modelName, err.Error()))
		return nil, false
	}
	if channel == nil {
		RenderSeedanceNativeError(c, http.StatusNotFound, "InvalidEndpointOrModel.NotFound", fmt.Sprintf("分组 %s 下模型 %s 无可用渠道", usingGroup, modelName))
		return nil, false
	}
	return channel, true
}

// BuildSeedanceNativeTaskResponse 将本地 task 渲染为 BytePlus / ModelArk native task object。
// 只使用 public task id 和 canonical Task.Data，不暴露上游真实 task id。
func BuildSeedanceNativeTaskResponse(task *model.Task) (*seedanceNativeTaskResponse, error) {
	var data seedanceCanonicalTaskData
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &data); err != nil {
			return nil, err
		}
	}
	modelName := task.Properties.OriginModelName
	if modelName == "" {
		modelName = data.Model
	}
	createdAt := task.CreatedAt
	if data.CreatedAt > 0 {
		createdAt = data.CreatedAt.Unix()
	}
	updatedAt := task.UpdatedAt
	if data.UpdatedAt > 0 {
		updatedAt = data.UpdatedAt.Unix()
	}
	resp := &seedanceNativeTaskResponse{
		ID:              task.TaskID,
		Model:           modelName,
		Status:          toSeedanceNativeStatus(task.Status),
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		Content:         data.Content,
		Seed:            data.Seed,
		Resolution:      data.Request.Resolution,
		Ratio:           data.Request.Ratio,
		Duration:        data.Request.Duration,
		FramesPerSecond: data.FramesPerSecond,
		GenerateAudio:   data.Request.GenerateAudio,
		Draft:           data.Draft,
		ServiceTier:     data.ServiceTier,
		Usage: &seedanceNativeTaskUsage{
			CompletionTokens: data.Usage.CompletionTokens,
			TotalTokens:      data.Usage.TotalTokens,
		},
	}
	if resp.Resolution == "" {
		resp.Resolution = data.Resolution
	}
	if resp.Ratio == "" {
		resp.Ratio = data.Ratio
	}
	if resp.Duration == 0 {
		resp.Duration = data.Duration
	}
	if resp.GenerateAudio == nil {
		resp.GenerateAudio = data.GenerateAudio
	}
	if resp.ExecutionExpiresAfter == 0 {
		resp.ExecutionExpiresAfter = data.ExecutionExpiresAfter
	}
	if resp.Priority == 0 {
		resp.Priority = data.Priority
	}
	if resp.ExecutionExpiresAfter == 0 {
		resp.ExecutionExpiresAfter = data.Request.ExecutionExpiresAfter
	}
	if resp.Priority == 0 {
		resp.Priority = data.Request.Priority
	}
	if resp.ServiceTier == "" {
		resp.ServiceTier = "default"
	}
	if resp.Content.VideoURL == "" {
		resp.Content.VideoURL = data.VideoURL
	}
	if resp.Content.LastFrameURL == "" {
		resp.Content.LastFrameURL = data.LastFrameURL
	}
	if resp.Content.VideoURL == "" {
		resp.Content.VideoURL = task.GetResultURL()
	}
	if task.Status == model.TaskStatusFailure || data.Error.Code != "" || data.Error.Message != "" {
		resp.Error = &seedanceNativeTaskError{
			Code:    data.Error.Code,
			Message: data.Error.Message,
		}
		if resp.Error.Message == "" {
			resp.Error.Message = task.FailReason
		}
	} else {
		resp.Error = nil
	}
	return resp, nil
}

// SeedanceNativeTaskGet 查询当前用户单个 Seedance native task。
// 只按 public task id 和当前 token 用户查询，非本人、非准入 channel 或不存在统一返回 native 404。
func SeedanceNativeTaskGet(c *gin.Context) {
	if !seedanceNativeTokenAuth(c) {
		return
	}
	taskID := c.Param("task_id")
	if taskID == "" {
		taskID = c.Param("id")
	}
	task, exist, err := model.GetByTaskId(c.GetInt("id"), taskID)
	if err != nil {
		RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", "failed to get task")
		return
	}
	if !exist || !IsSeedanceNativeRenderableTask(task) {
		RenderSeedanceNativeError(c, http.StatusNotFound, "ResourceNotFound.Task", "task not found")
		return
	}
	resp, err := BuildSeedanceNativeTaskResponse(task)
	if err != nil {
		RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", "failed to render task")
		return
	}
	c.JSON(http.StatusOK, resp)
}

// SeedanceNativeTaskList 列出当前用户最近 7 天 Seedance native 可渲染任务。
// 查询层先按用户、平台、时间和可表达过滤条件收敛，响应层继续保证只渲染已准入 channel。
func SeedanceNativeTaskList(c *gin.Context) {
	if !seedanceNativeTokenAuth(c) {
		return
	}
	pageNum, ok := seedanceQueryInt(c, "page_num", 1)
	if !ok || pageNum < 1 || pageNum > 500 {
		RenderSeedanceNativeError(c, http.StatusBadRequest, "InvalidParameter.InvalidValue", "page_num must be between 1 and 500")
		return
	}
	pageSize, ok := seedanceQueryInt(c, "page_size", 10)
	if !ok || pageSize < 1 || pageSize > 500 {
		RenderSeedanceNativeError(c, http.StatusBadRequest, "InvalidParameter.InvalidValue", "page_size must be between 1 and 500")
		return
	}

	serviceTier := c.Query("filter.service_tier")
	if serviceTier != "" && serviceTier != "default" {
		RenderSeedanceNativeError(c, http.StatusBadRequest, "InvalidParameter.Unsupported", "service_tier is not configurable for Seedance 2.0")
		return
	}

	statuses, ok := seedanceInternalStatuses(c.Query("filter.status"))
	if !ok {
		RenderSeedanceNativeError(c, http.StatusBadRequest, "InvalidParameter.InvalidValue", "filter.status is invalid")
		return
	}
	if c.Query("filter.status") == "cancelled" {
		c.JSON(http.StatusOK, seedanceNativeTaskListResponse{
			Items: []*seedanceNativeTaskResponse{},
			Total: 0,
		})
		return
	}

	filtered := seedanceLoadFilteredNativeTasks(c.GetInt("id"), seedanceNativeListFilter{
		TaskIDs: c.QueryArray("filter.task_ids"),
		Model:   c.Query("filter.model"),
		Status:  statuses,
	})
	startIdx := (pageNum - 1) * pageSize
	if startIdx > len(filtered) {
		startIdx = len(filtered)
	}
	endIdx := startIdx + pageSize
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}
	items := make([]*seedanceNativeTaskResponse, 0, endIdx-startIdx)
	for _, task := range filtered[startIdx:endIdx] {
		item, err := BuildSeedanceNativeTaskResponse(task)
		if err != nil {
			RenderSeedanceNativeError(c, http.StatusInternalServerError, "InternalServiceError", "failed to render task")
			return
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, seedanceNativeTaskListResponse{
		Items: items,
		Total: int64(len(filtered)),
	})
}

func seedanceLoadFilteredNativeTasks(userID int, filter seedanceNativeListFilter) []*model.Task {
	params := model.SyncTaskQueryParams{
		StartTimestamp: time.Now().Add(-7 * 24 * time.Hour).Unix(),
	}
	const batchSize = 500
	filtered := make([]*model.Task, 0)
	for offset := 0; ; offset += batchSize {
		tasks := model.TaskGetAllUserTask(userID, offset, batchSize, params)
		if len(tasks) == 0 {
			break
		}
		filtered = append(filtered, seedanceFilterNativeTasks(tasks, filter)...)
		if len(tasks) < batchSize {
			break
		}
	}
	return filtered
}

type seedanceNativeListFilter struct {
	TaskIDs []string
	Model   string
	Status  []model.TaskStatus
}

func seedanceFilterNativeTasks(tasks []*model.Task, filter seedanceNativeListFilter) []*model.Task {
	taskIDSet := make(map[string]bool, len(filter.TaskIDs))
	for _, taskID := range filter.TaskIDs {
		if taskID != "" {
			taskIDSet[taskID] = true
		}
	}
	statusSet := make(map[model.TaskStatus]bool, len(filter.Status))
	for _, status := range filter.Status {
		statusSet[status] = true
	}
	filtered := make([]*model.Task, 0, len(tasks))
	for _, task := range tasks {
		if !IsSeedanceNativeRenderableTask(task) {
			continue
		}
		if len(taskIDSet) > 0 && !taskIDSet[task.TaskID] {
			continue
		}
		if filter.Model != "" && task.Properties.OriginModelName != filter.Model {
			continue
		}
		if len(statusSet) > 0 && !statusSet[task.Status] {
			continue
		}
		filtered = append(filtered, task)
	}
	return filtered
}

// IsSeedanceNativeRenderableTask 判断 task 是否可按 Seedance native contract 渲染。
// 当前只开放 DoubaoVideo / VolcEngine，其他兼容 channel 需单独完成 C1 验证。
func IsSeedanceNativeRenderableTask(task *model.Task) bool {
	if task == nil {
		return false
	}
	switch task.Platform {
	case constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeXRTokenArkVideo)):
		return true
	default:
		return false
	}
}

// RenderSeedanceNativeError 输出 Seedance native error shell。
func RenderSeedanceNativeError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
			"type":    seedanceNativeErrorType(status),
		},
	})
}

func toSeedanceNativeStatus(status model.TaskStatus) string {
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
		return "running"
	}
}

func seedanceInternalStatuses(status string) ([]model.TaskStatus, bool) {
	switch status {
	case "":
		return nil, true
	case "queued":
		return []model.TaskStatus{
			model.TaskStatusNotStart,
			model.TaskStatusSubmitted,
			model.TaskStatusQueued,
		}, true
	case "running":
		return []model.TaskStatus{model.TaskStatusInProgress}, true
	case "succeeded":
		return []model.TaskStatus{model.TaskStatusSuccess}, true
	case "failed":
		return []model.TaskStatus{model.TaskStatusFailure}, true
	case "cancelled":
		return []model.TaskStatus{}, true
	default:
		return nil, false
	}
}

func seedanceQueryInt(c *gin.Context, key string, defaultValue int) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func seedanceNativeErrorType(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BadRequest"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "NotFound"
	case http.StatusTooManyRequests:
		return "TooManyRequests"
	default:
		return "InternalServerError"
	}
}
