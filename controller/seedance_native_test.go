package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/doubao"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestConvertSeedanceNativeCreateRequestPreservesNativeFields(t *testing.T) {
	t.Parallel()

	nativeReq := seedanceNativeCreateRequest{
		Model: "seedance-2-0-pro",
		Content: []doubao.ContentItem{
			{Type: "text", Text: "city skyline at sunset"},
			{Type: "image_url", ImageURL: &doubao.MediaURL{URL: "https://example.com/frame.png"}},
		},
		ReturnLastFrame:       boolPtr(true),
		ServiceTier:           "default",
		ExecutionExpiresAfter: intPtr(7200),
		GenerateAudio:         boolPtr(false),
		Priority:              intPtr(7),
		Resolution:            "1080p",
		Ratio:                 "16:9",
		Duration:              intPtr(5),
		Watermark:             boolPtr(false),
	}

	taskReq, err := convertSeedanceNativeCreateRequest(nativeReq)
	require.NoError(t, err)

	assert.Equal(t, "seedance-2-0-pro", taskReq.Model)
	assert.Equal(t, "city skyline at sunset", taskReq.Prompt)
	assert.Equal(t, "5", taskReq.Seconds)
	assert.Equal(t, nativeReq.Content, taskReq.Metadata["content"])
	assert.Equal(t, true, taskReq.Metadata["return_last_frame"])
	assert.Equal(t, "default", taskReq.Metadata["service_tier"])
	assert.Equal(t, 7200, taskReq.Metadata["execution_expires_after"])
	assert.Equal(t, false, taskReq.Metadata["generate_audio"])
	assert.Equal(t, 7, taskReq.Metadata["priority"])
	assert.Equal(t, "1080p", taskReq.Metadata["resolution"])
	assert.Equal(t, "16:9", taskReq.Metadata["ratio"])
	assert.Equal(t, 5, taskReq.Metadata["duration"])
	assert.Equal(t, false, taskReq.Metadata["watermark"])
}

func TestConvertSeedanceNativeCreateRequestRejectsUnsupportedFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mut  func(*seedanceNativeCreateRequest)
		want string
	}{
		{
			name: "callback_url",
			mut:  func(req *seedanceNativeCreateRequest) { req.CallbackURL = "https://example.com/callback" },
			want: "callback_url",
		},
		{
			name: "service_tier_flex",
			mut:  func(req *seedanceNativeCreateRequest) { req.ServiceTier = "flex" },
			want: "service_tier",
		},
		{
			name: "frames",
			mut:  func(req *seedanceNativeCreateRequest) { req.Frames = intPtr(120) },
			want: "frames",
		},
		{
			name: "seed",
			mut:  func(req *seedanceNativeCreateRequest) { req.Seed = intPtr(123) },
			want: "seed",
		},
		{
			name: "camera_fixed",
			mut:  func(req *seedanceNativeCreateRequest) { req.CameraFixed = boolPtr(true) },
			want: "camera_fixed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := seedanceNativeCreateRequest{
				Model: "seedance-2-0-pro",
				Content: []doubao.ContentItem{
					{Type: "text", Text: "city skyline"},
				},
			}
			tt.mut(&req)

			_, err := convertSeedanceNativeCreateRequest(req)
			require.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), tt.want), "error %q should mention %q", err.Error(), tt.want)
		})
	}
}

func TestSeedanceNativeCreateErrorCodesMatchContract(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
		wantErr  string
		wantType string
	}{
		{
			name:     "invalid_json",
			body:     `{"model":`,
			wantCode: http.StatusBadRequest,
			wantErr:  "InvalidParameter.InvalidJSON",
			wantType: "BadRequest",
		},
		{
			name:     "callback_url",
			body:     `{"model":"seedance-2-0-pro","content":[{"type":"text","text":"city skyline"}],"callback_url":"https://example.com/callback"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "OperationDenied.CallbackNotSupported",
			wantType: "BadRequest",
		},
		{
			name:     "unsupported_service_tier",
			body:     `{"model":"seedance-2-0-pro","content":[{"type":"text","text":"city skyline"}],"service_tier":"flex"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "InvalidParameter.Unsupported",
			wantType: "BadRequest",
		},
		{
			name:     "unsupported_seed",
			body:     `{"model":"seedance-2-0-pro","content":[{"type":"text","text":"city skyline"}],"seed":123}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "InvalidParameter.Unsupported",
			wantType: "BadRequest",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/api/v3/contents/generations/tasks", SeedanceNativeCreate)

			w := httptest.NewRecorder()
			req := httptestNewJSONRequest(tt.body)
			router.ServeHTTP(w, req)

			require.Equal(t, tt.wantCode, w.Code)
			assertSeedanceNativeError(t, w.Body.Bytes(), tt.wantErr, tt.wantType)
		})
	}
}

func TestPrepareSeedanceNativeCreateRequestReplacesReusableBody(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Request = httptestNewJSONRequest(`{
		"model":"seedance-2-0-pro",
		"content":[{"type":"text","text":"city skyline"}],
		"duration":5,
		"ratio":"16:9"
	}`)
	_, err := common.GetBodyStorage(c)
	require.NoError(t, err)

	require.NoError(t, prepareSeedanceNativeCreateRequest(c))

	assert.True(t, c.GetBool("seedance_native_response"))
	storage, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	body, err := storage.Bytes()
	require.NoError(t, err)

	var taskReq relaycommon.TaskSubmitReq
	require.NoError(t, common.Unmarshal(body, &taskReq))
	assert.Equal(t, "seedance-2-0-pro", taskReq.Model)
	assert.Equal(t, "city skyline", taskReq.Prompt)
	assert.Equal(t, "5", taskReq.Seconds)
	assert.Equal(t, float64(5), taskReq.Metadata["duration"])
	assert.Equal(t, "16:9", taskReq.Metadata["ratio"])
}

func TestSeedanceNativeCreateRejectsNonSeedanceChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v3/contents/generations/tasks", func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeSora)
		SeedanceNativeCreate(c)
	})

	w := httptest.NewRecorder()
	req := httptestNewJSONRequest(`{
		"model":"seedance-2-0-pro",
		"content":[{"type":"text","text":"city skyline"}]
	}`)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assertSeedanceNativeError(t, w.Body.Bytes(), "InvalidEndpointOrModel.NotFound", "NotFound")
}

func TestBuildSeedanceNativeTaskResponseUsesPublicIDAndCanonicalData(t *testing.T) {
	t.Parallel()

	task := &model.Task{
		TaskID:     "public_task_123",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		CreatedAt:  1710000000,
		UpdatedAt:  1710000100,
		SubmitTime: 1710000000,
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream_task_123",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"model":"upstream-seedance",
			"status":"succeeded",
			"content":{"video_url":"https://cdn.example.com/video.mp4"},
			"resolution":"1080p",
			"duration":5,
			"ratio":"16:9",
			"service_tier":"default",
			"usage":{"completion_tokens":10,"total_tokens":10},
			"request":{"generate_audio":true}
		}`),
	}

	nativeTask, err := buildSeedanceNativeTaskResponse(task)
	require.NoError(t, err)

	assert.Equal(t, "public_task_123", nativeTask.ID)
	assert.Equal(t, "seedance-2-0-pro", nativeTask.Model)
	assert.Equal(t, "succeeded", nativeTask.Status)
	assert.Equal(t, "https://cdn.example.com/video.mp4", nativeTask.Content.VideoURL)
	assert.Equal(t, "1080p", nativeTask.Resolution)
	assert.Equal(t, 5, nativeTask.Duration)
	assert.Equal(t, "16:9", nativeTask.Ratio)
	assert.Equal(t, "default", nativeTask.ServiceTier)
	assert.Equal(t, 10, nativeTask.Usage.TotalTokens)
	assert.Equal(t, true, nativeTask.Request["generate_audio"])
}

func TestBuildSeedanceNativeListQueryMapsFiltersToTaskQuery(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks?page_num=2&page_size=20&filter.status=succeeded&filter.task_ids=task_a&filter.task_ids=task_b&filter.model=seedance-2-0-pro&filter.service_tier=default", nil)

	pageNum, pageSize, query, err := buildSeedanceNativeListQuery(c, 1710000000)
	require.NoError(t, err)

	assert.Equal(t, 2, pageNum)
	assert.Equal(t, 20, pageSize)
	assert.Equal(t, "SUCCESS", query.Status)
	assert.ElementsMatch(t, []string{"task_a", "task_b"}, query.TaskIDs)
	assert.Equal(t, "generate", query.Action)
	assert.EqualValues(t, 1710000000-7*24*3600, query.StartTimestamp)
	assert.EqualValues(t, 1710000000, query.EndTimestamp)
	assert.Equal(t, "seedance-2-0-pro", query.Model)
}

func TestBuildSeedanceNativeListQueryRejectsInvalidPagination(t *testing.T) {
	t.Parallel()

	tests := []string{
		"page_num=0",
		"page_num=501",
		"page_num=abc",
		"page_size=0",
		"page_size=501",
		"page_size=abc",
	}

	for _, query := range tests {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(nil)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks?"+query, nil)

			_, _, _, err := buildSeedanceNativeListQuery(c, 1710000000)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "page_")
		})
	}
}

func TestSeedanceNativeListRejectsInvalidStatusWithNativeBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v3/contents/generations/tasks", func(c *gin.Context) {
		c.Set("id", 11)
		SeedanceNativeList(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks?filter.status=expired", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assertSeedanceNativeError(t, w.Body.Bytes(), "InvalidParameter.InvalidValue", "BadRequest")
}

func TestSeedanceNativeListReturnsTotalBeyondFirstFiveHundredRows(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)

	for i := 0; i < 501; i++ {
		task := newSeedanceNativeTestTask(fmt.Sprintf("task_sora_%03d", i), 11, constant.ChannelTypeSora, "seedance-2-0-pro")
		task.SubmitTime = seedanceNativeListNow() + int64(i)
		insertSeedanceNativeTask(t, task)
	}
	seedanceTask := newSeedanceNativeTestTask("task_seedance_after_500", 11, constant.ChannelTypeDoubaoVideo, "seedance-2-0-pro")
	seedanceTask.SubmitTime = seedanceNativeListNow() - 1
	insertSeedanceNativeTask(t, seedanceTask)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v3/contents/generations/tasks", func(c *gin.Context) {
		c.Set("id", 11)
		SeedanceNativeList(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks?page_num=1&page_size=10&filter.model=seedance-2-0-pro", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got struct {
		Items []seedanceNativeTaskResponse `json:"items"`
		Total int                          `json:"total"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, "task_seedance_after_500", got.Items[0].ID)
}

func TestSeedanceNativeGetReturnsOnlyUserSeedanceTask(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)

	seedanceTask := newSeedanceNativeTestTask("task_public", 11, constant.ChannelTypeDoubaoVideo, "seedance-2-0-pro")
	insertSeedanceNativeTask(t, seedanceTask)
	insertSeedanceNativeTask(t, newSeedanceNativeTestTask("task_public", 12, constant.ChannelTypeDoubaoVideo, "seedance-2-0-pro"))
	insertSeedanceNativeTask(t, newSeedanceNativeTestTask("task_sora", 11, constant.ChannelTypeSora, "sora"))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v3/contents/generations/tasks/:id", func(c *gin.Context) {
		c.Set("id", 11)
		SeedanceNativeGet(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks/task_public", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got seedanceNativeTaskResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "task_public", got.ID)
	assert.Equal(t, "seedance-2-0-pro", got.Model)
	assert.Equal(t, "https://cdn.example.com/task_public.mp4", got.Content.VideoURL)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks/task_sora", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "ResourceNotFound.Task")
}

func TestSeedanceNativeListUsesUserTaskSourceAndFiltersModel(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)

	insertSeedanceNativeTask(t, newSeedanceNativeTestTask("task_keep", 11, constant.ChannelTypeDoubaoVideo, "seedance-2-0-pro"))
	insertSeedanceNativeTask(t, newSeedanceNativeTestTask("task_other_model", 11, constant.ChannelTypeDoubaoVideo, "seedance-2-0-lite"))
	insertSeedanceNativeTask(t, newSeedanceNativeTestTask("task_sora", 11, constant.ChannelTypeSora, "seedance-2-0-pro"))
	insertSeedanceNativeTask(t, newSeedanceNativeTestTask("task_other_user", 12, constant.ChannelTypeDoubaoVideo, "seedance-2-0-pro"))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v3/contents/generations/tasks", func(c *gin.Context) {
		c.Set("id", 11)
		SeedanceNativeList(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks?filter.model=seedance-2-0-pro&page_num=1&page_size=10", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got struct {
		Items []seedanceNativeTaskResponse `json:"items"`
		Total int                          `json:"total"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, "task_keep", got.Items[0].ID)
}

func TestSeedanceNativeOpenAICrossProtocolRenderersUsePublicID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		task *model.Task
	}{
		{
			name: "openai_create_native_and_openai_get",
			task: newSeedanceNativeRenderableTask("task_openai_created", "upstream_openai_created", "seedance-2-0-pro",
				"https://cdn.example.com/openai-created.mp4"),
		},
		{
			name: "native_create_native_and_openai_get",
			task: newSeedanceNativeRenderableTask("task_native_created", "upstream_native_created", "seedance-2-0-pro",
				"https://cdn.example.com/native-created.mp4"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nativeResp, err := buildSeedanceNativeTaskResponse(tt.task)
			require.NoError(t, err)
			assert.Equal(t, tt.task.TaskID, nativeResp.ID)
			assert.Equal(t, tt.task.Properties.OriginModelName, nativeResp.Model)
			assert.Equal(t, "succeeded", nativeResp.Status)
			assert.NotContains(t, nativeResp.Content.VideoURL, tt.task.PrivateData.UpstreamTaskID)

			openAIBody, err := (&doubao.TaskAdaptor{}).ConvertToOpenAIVideo(tt.task)
			require.NoError(t, err)
			var openAIResp dto.OpenAIVideo
			require.NoError(t, common.Unmarshal(openAIBody, &openAIResp))
			assert.Equal(t, tt.task.TaskID, openAIResp.ID)
			assert.Equal(t, tt.task.TaskID, openAIResp.TaskID)
			assert.Equal(t, dto.VideoStatusCompleted, openAIResp.Status)
			assert.Equal(t, nativeResp.Content.VideoURL, openAIResp.Metadata["url"])
			assert.NotContains(t, string(openAIBody), tt.task.PrivateData.UpstreamTaskID)
		})
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func httptestNewJSONRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v3/contents/generations/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func setupSeedanceNativeControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Task{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func insertSeedanceNativeTask(t *testing.T, task *model.Task) {
	t.Helper()
	require.NoError(t, model.DB.Create(task).Error)
}

func assertSeedanceNativeError(t *testing.T, body []byte, code string, errType string) {
	t.Helper()

	var got struct {
		Error struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	require.NoError(t, common.Unmarshal(body, &got))
	assert.Equal(t, code, got.Error.Code)
	assert.Equal(t, errType, got.Error.Type)
}

func newSeedanceNativeRenderableTask(taskID string, upstreamTaskID string, originModel string, videoURL string) *model.Task {
	data := `{
		"id":"` + taskID + `",
		"model":"` + originModel + `",
		"status":"succeeded",
		"content":{"video_url":"` + videoURL + `"},
		"service_tier":"default",
		"request":{"duration":5,"service_tier":"default"}
	}`
	return &model.Task{
		TaskID:     taskID,
		Platform:   constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
		Action:     constant.TaskActionGenerate,
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		CreatedAt:  1710000000,
		UpdatedAt:  1710000100,
		Properties: model.Properties{OriginModelName: originModel},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: upstreamTaskID,
		},
		Data: []byte(data),
	}
}

func newSeedanceNativeTestTask(taskID string, userID int, channelType int, originModel string) *model.Task {
	data := `{
		"id":"upstream_` + taskID + `",
		"model":"` + originModel + `",
		"status":"succeeded",
		"content":{"video_url":"https://cdn.example.com/` + taskID + `.mp4"},
		"service_tier":"default"
	}`
	return &model.Task{
		TaskID:     taskID,
		UserId:     userID,
		Platform:   constant.TaskPlatform(strconv.Itoa(channelType)),
		Action:     constant.TaskActionGenerate,
		Status:     model.TaskStatusSuccess,
		SubmitTime: seedanceNativeListNow(),
		Properties: model.Properties{OriginModelName: originModel},
		Data:       []byte(data),
	}
}
