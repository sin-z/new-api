package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSeedanceNativeBuildOpenAIRequestKeepsNativeFieldsInMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v3/contents/generations/tasks", strings.NewReader(`{
		"model":"seedance-2-0-pro",
		"content":[
			{"type":"text","text":"A quiet street at night"},
			{"type":"image_url","image_url":{"url":"https://example.com/ref.png"}}
		],
		"resolution":"1080p",
		"ratio":"16:9",
		"duration":5,
		"generate_audio":true,
		"service_tier":"default",
		"execution_expires_after":172800,
		"watermark":false
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, taskReq, ok := seedanceNativeBuildOpenAIRequest(ctx)

	require.True(t, ok)
	require.False(t, ctx.IsAborted())
	require.Equal(t, "seedance-2-0-pro", taskReq.Model)
	require.Equal(t, "A quiet street at night", taskReq.Prompt)
	require.Zero(t, taskReq.Duration)
	require.Empty(t, taskReq.Seconds)
	require.Equal(t, "1080p", taskReq.Metadata["resolution"])
	require.EqualValues(t, 5, taskReq.Metadata["duration"])
	contentRaw, ok := taskReq.Metadata["content"].([]interface{})
	require.True(t, ok)
	require.Len(t, contentRaw, 2)
}

func TestSeedanceNativeRewriteRequestSetsOpenAIVideoBodyInHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v3/contents/generations/tasks", strings.NewReader(`{"model":"seedance-2-0-pro"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ok := seedanceNativeRewriteRequest(ctx, relaycommon.TaskSubmitReq{
		Prompt: "prompt",
		Model:  "seedance-2-0-pro",
		Metadata: map[string]interface{}{
			"ratio": "16:9",
		},
	})

	require.True(t, ok)
	require.Equal(t, "/v1/video/generations", ctx.Request.URL.Path)
	require.True(t, ctx.GetBool("seedance_native_response"))
	var rewritten relaycommon.TaskSubmitReq
	require.NoError(t, common.UnmarshalBodyReusable(ctx, &rewritten))
	require.Equal(t, "seedance-2-0-pro", rewritten.Model)
	require.Equal(t, "prompt", rewritten.Prompt)
	require.Equal(t, "16:9", rewritten.Metadata["ratio"])
}

func TestSeedanceNativeBuildOpenAIRequestRejectsUnsupportedServiceTier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v3/contents/generations/tasks", strings.NewReader(`{
		"model":"seedance-2-0-pro",
		"content":[{"type":"text","text":"prompt"}],
		"service_tier":"flex"
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, _, ok := seedanceNativeBuildOpenAIRequest(ctx)

	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "InvalidParameter.Unsupported")
}

func TestSeedanceNativeBuildOpenAIRequestRejectsInvalidDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v3/contents/generations/tasks", strings.NewReader(`{
		"model":"seedance-2-0-pro",
		"content":[{"type":"text","text":"prompt"}],
		"duration":1
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, _, ok := seedanceNativeBuildOpenAIRequest(ctx)

	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "InvalidParameter.InvalidValue")
	require.Contains(t, recorder.Body.String(), "duration")
}

func TestBuildSeedanceNativeTaskResponseUsesPublicIDAndCanonicalData(t *testing.T) {
	t.Parallel()

	task := &model.Task{
		TaskID:    "task_public_123",
		Status:    model.TaskStatusSuccess,
		CreatedAt: time.Unix(1710000000, 0).Unix(),
		UpdatedAt: time.Unix(1710000100, 0).Unix(),
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"model":"volcengine/seedance-2-0-pro",
			"status":"succeeded",
			"content":{"video_url":"https://cdn.example.com/video.mp4"},
			"usage":{"completion_tokens":1000,"total_tokens":1000},
			"request":{"resolution":"1080p","ratio":"16:9","duration":5,"generate_audio":true},
			"service_tier":"default",
			"created_at":1710000000,
			"updated_at":1710000100
		}`),
	}

	resp, err := BuildSeedanceNativeTaskResponse(task)
	if err != nil {
		t.Fatalf("BuildSeedanceNativeTaskResponse returned error: %v", err)
	}
	if resp.ID != "task_public_123" {
		t.Fatalf("id = %q, want public id", resp.ID)
	}
	if resp.Content.VideoURL != "https://cdn.example.com/video.mp4" {
		t.Fatalf("video_url = %q, want canonical video url", resp.Content.VideoURL)
	}
	if resp.Model != "seedance-2-0-pro" {
		t.Fatalf("model = %q, want origin model", resp.Model)
	}
	if resp.Resolution != "1080p" || resp.Ratio != "16:9" || resp.Duration != 5 {
		b, _ := json.Marshal(resp)
		t.Fatalf("request snapshot fields missing: %s", b)
	}
}

func TestSeedanceCanonicalTaskDataOmitsEmptyNativeFields(t *testing.T) {
	t.Parallel()

	data := seedanceCanonicalTaskData{
		ID:     "upstream_task_123",
		Model:  "doubao-seedance-2-0-260128",
		Status: "queued",
	}

	body, err := json.Marshal(data)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"framespersecond"`)
	require.NotContains(t, string(body), `"draft"`)
	require.NotContains(t, string(body), `"last_frame_url"`)
}

func TestRenderSeedanceTaskNotFoundUsesNativeErrorShell(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	RenderSeedanceNativeError(ctx, http.StatusNotFound, "ResourceNotFound.Task", "task not found")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if body := recorder.Body.String(); body == "" || body[0] != '{' {
		t.Fatalf("body = %q, want native JSON error shell", body)
	}
	var out struct {
		Error struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("error body is invalid JSON: %v", err)
	}
	if out.Error.Code != "ResourceNotFound.Task" || out.Error.Type != "NotFound" {
		t.Fatalf("error = %#v, want ResourceNotFound.Task/NotFound", out.Error)
	}
}

func TestIsSeedanceNativeRenderableTaskAllowsSeedanceCompatibleChannels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		platform constant.TaskPlatform
		want     bool
	}{
		{name: "doubao video", platform: constant.TaskPlatform("54"), want: true},
		{name: "volcengine", platform: constant.TaskPlatform("45"), want: true},
		{name: "xrtoken ark video", platform: constant.TaskPlatform("101"), want: true},
		{name: "sora excluded", platform: constant.TaskPlatform("1"), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task := &model.Task{Platform: tc.platform}
			if got := IsSeedanceNativeRenderableTask(task); got != tc.want {
				t.Fatalf("IsSeedanceNativeRenderableTask(%q) = %v, want %v", tc.platform, got, tc.want)
			}
		})
	}
}

func TestSeedanceNativeTaskGetReturnsNativeObjectForOwner(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_public_123",
		UserId:     10,
		Platform:   constant.TaskPlatform("54"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: time.Now().Unix(),
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream_task_123",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"model":"volcengine/seedance-2-0-pro",
			"status":"succeeded",
			"content":{"video_url":"https://cdn.example.com/video.mp4"},
			"usage":{"completion_tokens":1000,"total_tokens":1000},
			"resolution":"1080p",
			"ratio":"16:9",
			"duration":5,
			"service_tier":"default"
		}`),
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks/task_public_123", nil)
	ctx.Params = gin.Params{{Key: "task_id", Value: "task_public_123"}}
	ctx.Set("id", 10)

	SeedanceNativeTaskGet(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "upstream_task_123")
	var resp seedanceNativeTaskResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, "task_public_123", resp.ID)
	require.Equal(t, "succeeded", resp.Status)
	require.Equal(t, "https://cdn.example.com/video.mp4", resp.Content.VideoURL)
	require.Equal(t, "1080p", resp.Resolution)
	require.Equal(t, "16:9", resp.Ratio)
	require.Equal(t, 5, resp.Duration)
}

func TestSeedanceNativeTaskGetReadsXRTokenTopLevelVideoURL(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_public_123",
		UserId:     10,
		Platform:   constant.TaskPlatform("101"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: time.Now().Unix(),
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-260128",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream_task_123",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"model":"volcengine/doubao-seedance-2-0-260128",
			"status":"succeeded",
			"video_url":"https://cdn.example.com/xrtoken.mp4",
			"created_at":"2026-07-07T02:40:14Z",
			"updated_at":"2026-07-07T02:41:14Z"
		}`),
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks/task_public_123", nil)
	ctx.Params = gin.Params{{Key: "task_id", Value: "task_public_123"}}
	ctx.Set("id", 10)

	SeedanceNativeTaskGet(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "upstream_task_123")
	var resp seedanceNativeTaskResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, "task_public_123", resp.ID)
	require.Equal(t, "succeeded", resp.Status)
	require.Equal(t, "https://cdn.example.com/xrtoken.mp4", resp.Content.VideoURL)
	require.EqualValues(t, 1783392014, resp.CreatedAt)
	require.EqualValues(t, 1783392074, resp.UpdatedAt)
}

func TestSeedanceNativeTaskGetRendersXRTokenUsageAndNativeFields(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_public_123",
		UserId:     10,
		Platform:   constant.TaskPlatform("101"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: time.Now().Unix(),
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-260128",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream_task_123",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"model":"volcengine/doubao-seedance-2-0-260128",
			"status":"succeeded",
			"video_url":"https://cdn.example.com/xrtoken.mp4",
			"last_frame_url":"https://cdn.example.com/last-frame.png",
			"usage":{"completion_tokens":108900,"total_tokens":108900},
			"created_at":1779348818,
			"updated_at":1779348874,
			"seed":78674,
			"resolution":"720p",
			"ratio":"16:9",
			"duration":5,
			"framespersecond":24,
			"service_tier":"default",
			"execution_expires_after":172800,
			"generate_audio":true,
			"draft":false,
			"priority":0
		}`),
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks/task_public_123", nil)
	ctx.Params = gin.Params{{Key: "task_id", Value: "task_public_123"}}
	ctx.Set("id", 10)

	SeedanceNativeTaskGet(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "upstream_task_123")
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Status  string `json:"status"`
		Content struct {
			VideoURL     string `json:"video_url"`
			LastFrameURL string `json:"last_frame_url"`
		} `json:"content"`
		Usage struct {
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		CreatedAt             int64  `json:"created_at"`
		UpdatedAt             int64  `json:"updated_at"`
		Seed                  int    `json:"seed"`
		Resolution            string `json:"resolution"`
		Ratio                 string `json:"ratio"`
		Duration              int    `json:"duration"`
		FramesPerSecond       int    `json:"framespersecond"`
		ServiceTier           string `json:"service_tier"`
		ExecutionExpiresAfter int    `json:"execution_expires_after"`
		GenerateAudio         bool   `json:"generate_audio"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, "task_public_123", resp.ID)
	require.Equal(t, "doubao-seedance-2-0-260128", resp.Model)
	require.Equal(t, "succeeded", resp.Status)
	require.Equal(t, "https://cdn.example.com/xrtoken.mp4", resp.Content.VideoURL)
	require.Equal(t, "https://cdn.example.com/last-frame.png", resp.Content.LastFrameURL)
	require.Equal(t, 108900, resp.Usage.CompletionTokens)
	require.Equal(t, 108900, resp.Usage.TotalTokens)
	require.EqualValues(t, 1779348818, resp.CreatedAt)
	require.EqualValues(t, 1779348874, resp.UpdatedAt)
	require.Equal(t, 78674, resp.Seed)
	require.Equal(t, "720p", resp.Resolution)
	require.Equal(t, "16:9", resp.Ratio)
	require.Equal(t, 5, resp.Duration)
	require.Equal(t, 24, resp.FramesPerSecond)
	require.Equal(t, "default", resp.ServiceTier)
	require.Equal(t, 172800, resp.ExecutionExpiresAfter)
	require.True(t, resp.GenerateAudio)
	require.NotContains(t, recorder.Body.String(), `"draft"`)
	require.NotContains(t, recorder.Body.String(), `"priority"`)
}

func TestSeedanceNativeTaskGetHidesOtherUsersAndUnsupportedChannels(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_other_user",
		UserId:     11,
		Platform:   constant.TaskPlatform("54"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: time.Now().Unix(),
		Data:       []byte(`{"content":{"video_url":"https://cdn.example.com/other.mp4"}}`),
	})
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_sora",
		UserId:     10,
		Platform:   constant.TaskPlatform("1"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: time.Now().Unix(),
		Data:       []byte(`{"content":{"video_url":"https://cdn.example.com/sora.mp4"}}`),
	})

	for _, taskID := range []string{"task_other_user", "task_sora"} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks/"+taskID, nil)
		ctx.Params = gin.Params{{Key: "task_id", Value: taskID}}
		ctx.Set("id", 10)

		SeedanceNativeTaskGet(ctx)

		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Contains(t, recorder.Body.String(), "ResourceNotFound.Task")
	}
}

func TestSeedanceNativeTaskListFiltersRecentRenderableTasks(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)
	now := time.Now().Unix()
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_keep",
		UserId:     10,
		Platform:   constant.TaskPlatform("54"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: now,
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		Data: []byte(`{"content":{"video_url":"https://cdn.example.com/keep.mp4"},"service_tier":"default"}`),
	})
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_other_platform",
		UserId:     10,
		Platform:   constant.TaskPlatform("1"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: now,
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		Data: []byte(`{"content":{"video_url":"https://cdn.example.com/xrtoken.mp4"},"service_tier":"default"}`),
	})
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_old",
		UserId:     10,
		Platform:   constant.TaskPlatform("54"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: now - 8*24*60*60,
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		Data: []byte(`{"content":{"video_url":"https://cdn.example.com/old.mp4"},"service_tier":"default"}`),
	})
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_other_user",
		UserId:     11,
		Platform:   constant.TaskPlatform("54"),
		Status:     model.TaskStatusSuccess,
		SubmitTime: now,
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		Data: []byte(`{"content":{"video_url":"https://cdn.example.com/other.mp4"},"service_tier":"default"}`),
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks?page_num=1&page_size=10&filter.status=succeeded&filter.model=seedance-2-0-pro&filter.task_ids=task_keep&filter.task_ids=task_other_platform", nil)
	ctx.Set("id", 10)

	SeedanceNativeTaskList(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Items []seedanceNativeTaskResponse `json:"items"`
		Total int64                        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.EqualValues(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "task_keep", resp.Items[0].ID)
}

func TestSeedanceNativeTaskListRejectsUnsupportedServiceTier(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks?filter.service_tier=flex", nil)
	ctx.Set("id", 10)

	SeedanceNativeTaskList(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "InvalidParameter.Unsupported")
}

func TestSeedanceNativeTaskListCancelledFilterDoesNotReturnFailures(t *testing.T) {
	setupSeedanceNativeControllerTestDB(t)
	insertSeedanceNativeControllerTask(t, &model.Task{
		TaskID:     "task_failed",
		UserId:     10,
		Platform:   constant.TaskPlatform("54"),
		Status:     model.TaskStatusFailure,
		SubmitTime: time.Now().Unix(),
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		Data: []byte(`{"error":{"message":"upstream failed"},"service_tier":"default"}`),
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks?filter.status=cancelled", nil)
	ctx.Set("id", 10)

	SeedanceNativeTaskList(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Items []seedanceNativeTaskResponse `json:"items"`
		Total int64                        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.EqualValues(t, 0, resp.Total)
	require.Empty(t, resp.Items)
}

func TestRenderSeedanceTaskErrorUsesNativeShell(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v3/contents/generations/tasks", nil)

	renderSeedanceTaskError(ctx, &dto.TaskError{
		Code:       "invalid_request",
		Message:    "prompt is required",
		StatusCode: http.StatusBadRequest,
	})

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "invalid_request")
	require.Contains(t, recorder.Body.String(), "BadRequest")
}

func setupSeedanceNativeControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
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

func insertSeedanceNativeControllerTask(t *testing.T, task *model.Task) {
	t.Helper()
	if task.CreatedAt == 0 {
		task.CreatedAt = time.Now().Unix()
	}
	if task.UpdatedAt == 0 {
		task.UpdatedAt = task.CreatedAt
	}
	require.NoError(t, model.DB.Create(task).Error)
}
