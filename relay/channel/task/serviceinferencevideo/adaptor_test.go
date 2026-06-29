package serviceinferencevideo

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

func TestBuildRequestURLUsesVideoGeneratePath(t *testing.T) {
	t.Parallel()

	adaptor := newTestAdaptor("https://model.service-inference.ai", "sk-test")
	got, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{})
	if err != nil {
		t.Fatalf("BuildRequestURL returned error: %v", err)
	}

	want := "https://model.service-inference.ai/v1/video/generate"
	if got != want {
		t.Fatalf("BuildRequestURL() = %q, want %q", got, want)
	}
}

func TestFetchTaskUsesVideoTaskByIDPath(t *testing.T) {
	t.Parallel()

	adaptor := &TaskAdaptor{}
	_, err := adaptor.FetchTask("://bad-base", "sk-test", map[string]any{"task_id": "mvt-123"}, "")
	if err == nil || !strings.Contains(err.Error(), `/v1/video/tasks/mvt-123`) {
		t.Fatalf("FetchTask error = %v, want malformed URL containing service-inference.ai task path", err)
	}
}

func TestDoResponseReturnsWrappedUpstreamTaskIDAndPublicVideoID(t *testing.T) {
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
		Body: ioNopCloser(`{
			"task": {
				"id": "mvt-123",
				"status": "pending",
				"model": "dreamina-seedance-2-0-260128",
				"duration_seconds": 4,
				"outputs": [],
				"error": null,
				"created_at": "2026-05-26T05:26:52.505Z",
				"completed_at": null
			}
		}`),
	}

	taskID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	if taskErr != nil {
		t.Fatalf("DoResponse returned task error: %v", taskErr)
	}
	if taskID != "mvt-123" {
		t.Fatalf("taskID = %q, want wrapped upstream task id", taskID)
	}
	if !strings.Contains(string(taskData), `"id": "mvt-123"`) {
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
	if video.CreatedAt != 1779773212 {
		t.Fatalf("OpenAIVideo created_at = %d, want parsed upstream created_at", video.CreatedAt)
	}
}

func TestParseTaskResultMapsPendingCompletedFailedAndUnknown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		body         string
		wantStatus   model.TaskStatus
		wantProgress string
		wantURL      string
		wantReason   string
	}{
		{
			name:         "pending maps to queued",
			body:         `{"task":{"id":"mvt-123","status":"pending","outputs":[]}}`,
			wantStatus:   model.TaskStatusQueued,
			wantProgress: "10%",
		},
		{
			name:         "running maps to in progress",
			body:         `{"task":{"id":"mvt-123","status":"running","outputs":[]}}`,
			wantStatus:   model.TaskStatusInProgress,
			wantProgress: "50%",
		},
		{
			name: "completed maps output zero to success url and usage",
			body: `{
				"task": {
					"id": "mvt-123",
					"status": "completed",
					"outputs": ["https://cdn.example.com/main.mp4", "https://cdn.example.com/alt.mp4"],
					"usage": {"completion_tokens": 40594, "total_tokens": 40594}
				}
			}`,
			wantStatus:   model.TaskStatusSuccess,
			wantProgress: "100%",
			wantURL:      "https://cdn.example.com/main.mp4",
		},
		{
			name:         "failed maps error message",
			body:         `{"task":{"id":"mvt-123","status":"failed","error":{"code":"bad_request","message":"prompt rejected"}}}`,
			wantStatus:   model.TaskStatusFailure,
			wantProgress: "100%",
			wantReason:   "prompt rejected",
		},
		{
			name:         "unknown continues polling",
			body:         `{"task":{"id":"mvt-123","status":"provider_new_state","outputs":[]}}`,
			wantStatus:   model.TaskStatusInProgress,
			wantProgress: "30%",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			taskInfo, err := (&TaskAdaptor{}).ParseTaskResult([]byte(tt.body))
			if err != nil {
				t.Fatalf("ParseTaskResult returned error: %v", err)
			}
			if model.TaskStatus(taskInfo.Status) != tt.wantStatus {
				t.Fatalf("Status = %q, want %q", taskInfo.Status, tt.wantStatus)
			}
			if taskInfo.Progress != tt.wantProgress {
				t.Fatalf("Progress = %q, want %q", taskInfo.Progress, tt.wantProgress)
			}
			if taskInfo.Url != tt.wantURL {
				t.Fatalf("Url = %q, want %q", taskInfo.Url, tt.wantURL)
			}
			if taskInfo.Reason != tt.wantReason {
				t.Fatalf("Reason = %q, want %q", taskInfo.Reason, tt.wantReason)
			}
			if tt.wantURL != "" && (taskInfo.CompletionTokens != 40594 || taskInfo.TotalTokens != 40594) {
				t.Fatalf("usage = completion:%d total:%d, want 40594/40594", taskInfo.CompletionTokens, taskInfo.TotalTokens)
			}
		})
	}
}

func TestConvertToOpenAIVideoUsesOutputsAndDropsLastFrameURL(t *testing.T) {
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
			"task": {
				"id": "mvt-123",
				"status": "completed",
				"model": "dreamina-seedance-2-0-260128",
				"duration_seconds": 4,
				"outputs": ["https://cdn.example.com/main.mp4", "https://cdn.example.com/alt.mp4"],
				"last_frame_url": "https://model.service-inference.ai/v1/video/files/mvt-123/last-frame",
				"created_at": "2026-05-26T05:26:52.505Z",
				"completed_at": "2026-05-26T05:35:22.566Z",
				"usage": {"completion_tokens": 40594, "total_tokens": 40594}
			}
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
	if video.ID != "public_task_123" || video.TaskID != "public_task_123" {
		t.Fatalf("OpenAIVideo id/task_id = %q/%q, want public id", video.ID, video.TaskID)
	}
	if video.Model != "doubao-seedance-2-0-260128" {
		t.Fatalf("OpenAIVideo model = %q, want origin model", video.Model)
	}
	if video.Metadata["url"] != "https://cdn.example.com/main.mp4" {
		t.Fatalf("metadata.url = %#v, want outputs[0]", video.Metadata["url"])
	}
	if _, ok := video.Metadata["last_frame_url"]; ok {
		t.Fatalf("metadata.last_frame_url present, want dropped")
	}
	if video.Seconds != "4" {
		t.Fatalf("seconds = %q, want duration_seconds", video.Seconds)
	}
	if video.CreatedAt != 1779773212 {
		t.Fatalf("created_at = %d, want parsed upstream created_at", video.CreatedAt)
	}
	if video.CompletedAt != 1779773722 {
		t.Fatalf("completed_at = %d, want parsed upstream completed_at", video.CompletedAt)
	}
}

func TestConvertToOpenAIVideoFallsBackToTaskResultURL(t *testing.T) {
	t.Parallel()

	task := &model.Task{
		TaskID:    "public_task_123",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: time.Unix(1700000000, 0).Unix(),
		UpdatedAt: time.Unix(1700000100, 0).Unix(),
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://proxy.example.com/tasks/public_task_123",
		},
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-260128",
		},
		Data: []byte(`{"task":{"id":"mvt-123","status":"completed","outputs":[]}}`),
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
