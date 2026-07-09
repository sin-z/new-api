package xrtokenarkvideo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

func TestBuildRequestURLUsesV1TasksPath(t *testing.T) {
	t.Parallel()

	adaptor := newTestAdaptor("https://api.xrtoken.net", "sk-test")
	got, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{})
	if err != nil {
		t.Fatalf("BuildRequestURL returned error: %v", err)
	}

	want := "https://api.xrtoken.net/v1/contents/generations/tasks"
	if got != want {
		t.Fatalf("BuildRequestURL() = %q, want %q", got, want)
	}
}

func TestFetchTaskUsesVideoGenerationsPath(t *testing.T) {
	t.Parallel()

	adaptor := &TaskAdaptor{}
	_, err := adaptor.FetchTask("://bad-base", "sk-test", map[string]any{"task_id": "task_123"}, "")
	if err == nil || !strings.Contains(err.Error(), `/v1/videos/generations/task_123`) {
		t.Fatalf("FetchTask error = %v, want malformed URL containing XRToken video generation task path", err)
	}
}

func TestDoResponseReturnsUpstreamTaskIDAndPublicVideoID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-2-0-260128",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "public_task_123",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser(`{"id":"upstream_task_123","model":"volcengine/doubao-seedance-2-0-260128","status":"pending","created_at":1710000000}`),
	}

	taskID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	if taskErr != nil {
		t.Fatalf("DoResponse returned task error: %v", taskErr)
	}
	if taskID != "upstream_task_123" {
		t.Fatalf("taskID = %q, want upstream task id", taskID)
	}
	if !strings.Contains(string(taskData), `"id":"upstream_task_123"`) {
		t.Fatalf("taskData = %s, want raw upstream response", taskData)
	}

	var video dto.OpenAIVideo
	if err := json.Unmarshal(recorder.Body.Bytes(), &video); err != nil {
		t.Fatalf("response body is not OpenAIVideo: %v", err)
	}
	if video.ID != "public_task_123" || video.TaskID != "public_task_123" {
		t.Fatalf("OpenAIVideo id/task_id = %q/%q, want public id", video.ID, video.TaskID)
	}
	if video.Model != "doubao-seedance-2-0-260128" {
		t.Fatalf("OpenAIVideo model = %q, want origin model", video.Model)
	}
	if video.CreatedAt != 1710000000 {
		t.Fatalf("OpenAIVideo created_at = %d, want upstream created_at", video.CreatedAt)
	}
}

func TestDoResponseAcceptsStringTimestamps(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("seedance_native_response", true)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(`{
		"prompt":"prompt",
		"model":"doubao-seedance-2-0-260128",
		"metadata":{
			"ratio":"16:9",
			"duration":4,
			"generate_audio":true,
			"service_tier":"default"
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-2-0-260128",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "public_task_123",
		},
	}
	if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned task error: %v", taskErr)
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser(`{"id":"upstream_task_123","model":"volcengine/doubao-seedance-2-0-260128","status":"queued","created_at":"2026-07-07T02:40:14Z","updated_at":"2026-07-07T02:40:14Z"}`),
	}

	taskID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	if taskErr != nil {
		t.Fatalf("DoResponse returned task error: %v", taskErr)
	}
	if taskID != "upstream_task_123" {
		t.Fatalf("taskID = %q, want upstream task id", taskID)
	}
	if recorder.Body.String() != `{"id":"public_task_123"}` {
		t.Fatalf("response body = %s, want native public id body", recorder.Body.String())
	}
	var stored struct {
		ID          string `json:"id"`
		Ratio       string `json:"ratio"`
		Duration    int    `json:"duration"`
		ServiceTier string `json:"service_tier"`
	}
	if err := json.Unmarshal(taskData, &stored); err != nil {
		t.Fatalf("taskData is not JSON: %v", err)
	}
	if stored.ID != "upstream_task_123" || stored.Ratio != "16:9" || stored.Duration != 4 || stored.ServiceTier != "default" {
		t.Fatalf("taskData = %#v, want raw upstream data", stored)
	}
}

func TestParseTaskResultAcceptsStringTimestamps(t *testing.T) {
	t.Parallel()

	taskInfo, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"upstream_task_123",
		"status":"queued",
		"created_at":"2026-07-07T02:40:14Z",
		"updated_at":"2026-07-07T02:40:14Z"
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != model.TaskStatusQueued {
		t.Fatalf("Status = %q, want %q", taskInfo.Status, model.TaskStatusQueued)
	}
}

func TestParseTaskResultMapsUsageAndTopLevelVideoURL(t *testing.T) {
	t.Parallel()

	taskInfo, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"upstream_task_123",
		"status":"succeeded",
		"video_url":"https://cdn.example.com/video.mp4",
		"usage":{"completion_tokens":108900,"total_tokens":108900},
		"duration":5,
		"created_at":1710000000,
		"updated_at":1710000100
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}

	if taskInfo.Status != model.TaskStatusSuccess {
		t.Fatalf("Status = %q, want %q", taskInfo.Status, model.TaskStatusSuccess)
	}
	if taskInfo.Url != "https://cdn.example.com/video.mp4" {
		t.Fatalf("Url = %q, want top-level video_url", taskInfo.Url)
	}
	if taskInfo.CompletionTokens != 108900 || taskInfo.TotalTokens != 108900 {
		t.Fatalf("usage tokens = %d/%d, want 108900/108900", taskInfo.CompletionTokens, taskInfo.TotalTokens)
	}
}

func TestParseTaskResultReadsContentVideoURL(t *testing.T) {
	t.Parallel()

	taskInfo, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"upstream_task_123",
		"status":"succeeded",
		"content":{"video_url":"https://cdn.example.com/content-video.mp4"},
		"usage":{"completion_tokens":1000,"total_tokens":1000},
		"created_at":1710000000,
		"updated_at":1710000100
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}

	if taskInfo.Status != model.TaskStatusSuccess {
		t.Fatalf("Status = %q, want %q", taskInfo.Status, model.TaskStatusSuccess)
	}
	if taskInfo.Url != "https://cdn.example.com/content-video.mp4" {
		t.Fatalf("Url = %q, want content.video_url fallback", taskInfo.Url)
	}
}

func TestConvertToOpenAIVideoReadsTopLevelVideoURL(t *testing.T) {
	t.Parallel()

	task := &model.Task{
		TaskID:    "public_task_123",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: time.Unix(1700000000, 0).Unix(),
		UpdatedAt: time.Unix(1700000100, 0).Unix(),
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-260128",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"status":"succeeded",
			"video_url":"https://cdn.example.com/video.mp4",
			"duration":5,
			"created_at":1710000000,
			"updated_at":1710000100
		}`),
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	if err != nil {
		t.Fatalf("ConvertToOpenAIVideo returned error: %v", err)
	}

	var video dto.OpenAIVideo
	if err := json.Unmarshal(body, &video); err != nil {
		t.Fatalf("ConvertToOpenAIVideo returned invalid JSON: %v", err)
	}
	if video.Metadata["url"] != "https://cdn.example.com/video.mp4" {
		t.Fatalf("metadata.url = %#v, want top-level video_url", video.Metadata["url"])
	}
	if video.Seconds != "5" {
		t.Fatalf("seconds = %q, want upstream duration", video.Seconds)
	}
	if video.CreatedAt != 1710000000 {
		t.Fatalf("created_at = %d, want upstream created_at", video.CreatedAt)
	}
	if video.CompletedAt != 1710000100 {
		t.Fatalf("completed_at = %d, want upstream updated_at", video.CompletedAt)
	}
}

func TestConvertToOpenAIVideoReadsContentVideoURL(t *testing.T) {
	t.Parallel()

	task := &model.Task{
		TaskID:    "public_task_123",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: time.Unix(1700000000, 0).Unix(),
		UpdatedAt: time.Unix(1700000100, 0).Unix(),
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-260128",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"status":"succeeded",
			"content":{"video_url":"https://cdn.example.com/content-video.mp4"},
			"duration":5,
			"created_at":1710000000,
			"updated_at":1710000100
		}`),
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	if err != nil {
		t.Fatalf("ConvertToOpenAIVideo returned error: %v", err)
	}

	var video dto.OpenAIVideo
	if err := json.Unmarshal(body, &video); err != nil {
		t.Fatalf("ConvertToOpenAIVideo returned invalid JSON: %v", err)
	}
	if video.Metadata["url"] != "https://cdn.example.com/content-video.mp4" {
		t.Fatalf("metadata.url = %#v, want content.video_url", video.Metadata["url"])
	}
}

func TestConvertToOpenAIVideoFallsBackToTaskResultURL(t *testing.T) {
	t.Parallel()

	task := &model.Task{
		TaskID:    "public_task_123",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: time.Unix(1710000000, 0).Unix(),
		UpdatedAt: time.Unix(1710000100, 0).Unix(),
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-260128",
		},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://proxy.example.com/tasks/public_task_123",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"status":"succeeded",
			"video_url":"",
			"duration":5,
			"created_at":1710000000,
			"updated_at":1710000100
		}`),
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	if err != nil {
		t.Fatalf("ConvertToOpenAIVideo returned error: %v", err)
	}

	var video dto.OpenAIVideo
	if err := json.Unmarshal(body, &video); err != nil {
		t.Fatalf("ConvertToOpenAIVideo returned invalid JSON: %v", err)
	}
	if video.Metadata["url"] != "https://proxy.example.com/tasks/public_task_123" {
		t.Fatalf("metadata.url = %#v, want task result URL fallback", video.Metadata["url"])
	}
}

func newTestAdaptor(baseURL string, apiKey string) *TaskAdaptor {
	adaptor := &TaskAdaptor{}
	adaptor.Init(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: baseURL,
			ApiKey:         apiKey,
		},
	})
	return adaptor
}

type nopReadCloser struct {
	*strings.Reader
}

func (n nopReadCloser) Close() error {
	return nil
}

func ioNopCloser(body string) nopReadCloser {
	return nopReadCloser{Reader: strings.NewReader(body)}
}
