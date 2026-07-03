package doubao

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

func TestConvertToRequestPayloadKeepsNativeMetadataFields(t *testing.T) {
	t.Parallel()

	req := &relaycommon.TaskSubmitReq{
		Prompt:  "first prompt",
		Model:   "seedance-2-0-pro",
		Seconds: "9",
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": "first prompt"},
				map[string]interface{}{"type": "text", "text": "second prompt"},
				map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "https://example.com/ref.png"}},
			},
			"duration": 5,
			"priority": 3,
		},
	}

	payload, err := (&TaskAdaptor{}).convertToRequestPayload(req)
	if err != nil {
		t.Fatalf("convertToRequestPayload returned error: %v", err)
	}

	if payload.Duration == nil || int(*payload.Duration) != 5 {
		t.Fatalf("duration = %#v, want metadata duration 5", payload.Duration)
	}
	if payload.Priority == nil || int(*payload.Priority) != 3 {
		t.Fatalf("priority = %#v, want 3", payload.Priority)
	}
	if len(payload.Content) != 3 {
		t.Fatalf("content length = %d, want 3: %#v", len(payload.Content), payload.Content)
	}
	if payload.Content[0].Type != "text" || payload.Content[0].Text != "first prompt" {
		t.Fatalf("first content = %#v, want first text prompt", payload.Content[0])
	}
	if payload.Content[1].Type != "text" || payload.Content[1].Text != "second prompt" {
		t.Fatalf("second content = %#v, want second text prompt", payload.Content[1])
	}
	if payload.Content[2].Type != "image_url" || payload.Content[2].ImageURL == nil || payload.Content[2].ImageURL.URL != "https://example.com/ref.png" {
		t.Fatalf("third content = %#v, want image url", payload.Content[2])
	}
	body, err := common.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload returned error: %v", err)
	}
	if !strings.Contains(string(body), `"priority":3`) {
		t.Fatalf("payload body = %s, want priority in upstream JSON", body)
	}
}

func TestConvertToRequestPayloadBuildsLegacyOpenAIVideoContentWithoutNativeMetadata(t *testing.T) {
	t.Parallel()

	req := &relaycommon.TaskSubmitReq{
		Prompt:  "legacy prompt",
		Model:   "seedance-2-0-pro",
		Images:  []string{"https://example.com/ref.png"},
		Seconds: "9",
		Metadata: map[string]interface{}{
			"ratio": "16:9",
		},
	}

	payload, err := (&TaskAdaptor{}).convertToRequestPayload(req)
	if err != nil {
		t.Fatalf("convertToRequestPayload returned error: %v", err)
	}

	if payload.Duration == nil || int(*payload.Duration) != 9 {
		t.Fatalf("duration = %#v, want legacy seconds 9", payload.Duration)
	}
	if len(payload.Content) != 2 {
		t.Fatalf("content length = %d, want image + text: %#v", len(payload.Content), payload.Content)
	}
	if payload.Content[0].Type != "image_url" || payload.Content[0].ImageURL == nil || payload.Content[0].ImageURL.URL != "https://example.com/ref.png" {
		t.Fatalf("first content = %#v, want image url", payload.Content[0])
	}
	if payload.Content[1].Type != "text" || payload.Content[1].Text != "legacy prompt" {
		t.Fatalf("second content = %#v, want legacy prompt text", payload.Content[1])
	}
}

func TestDoResponseCanReturnNativeCreateBodyWithoutChangingTaskData(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("seedance_native_response", true)
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance-2-0-pro",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public_123",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser(`{"id":"upstream_task_123"}`),
	}

	upstreamID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(ctx, resp, info)
	if taskErr != nil {
		t.Fatalf("DoResponse returned task error: %v", taskErr)
	}
	if upstreamID != "upstream_task_123" {
		t.Fatalf("upstreamID = %q, want upstream_task_123", upstreamID)
	}
	if strings.Contains(string(taskData), "task_public_123") {
		t.Fatalf("taskData = %s, must not replace upstream raw id with public id", taskData)
	}
	var stored map[string]any
	if err := json.Unmarshal(taskData, &stored); err != nil {
		t.Fatalf("taskData is not JSON: %v", err)
	}
	if stored["id"] != "upstream_task_123" {
		t.Fatalf("taskData.id = %#v, want upstream_task_123", stored["id"])
	}
	if recorder.Body.String() != `{"id":"task_public_123"}` {
		t.Fatalf("response body = %s, want native public id body", recorder.Body.String())
	}
}

func TestDoResponseStoresNativeRequestSnapshotInTaskData(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("seedance_native_response", true)
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance-2-0-pro",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "volcengine/seedance-2-0-pro",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public_123",
		},
	}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(`{
		"prompt":"prompt",
		"model":"seedance-2-0-pro",
		"seconds":"5",
		"metadata":{
			"resolution":"1080p",
			"ratio":"16:9",
			"duration":5,
			"generate_audio":true,
			"service_tier":"default",
			"execution_expires_after":172800,
			"priority":3
		}
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info); taskErr != nil {
		t.Fatalf("ValidateRequestAndSetAction returned task error: %v", taskErr)
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser(`{"id":"upstream_task_123"}`),
	}

	_, taskData, taskErr := (&TaskAdaptor{}).DoResponse(ctx, resp, info)
	if taskErr != nil {
		t.Fatalf("DoResponse returned task error: %v", taskErr)
	}

	var stored struct {
		ID                    string `json:"id"`
		Model                 string `json:"model"`
		Status                string `json:"status"`
		Resolution            string `json:"resolution"`
		Ratio                 string `json:"ratio"`
		Duration              int    `json:"duration"`
		GenerateAudio         *bool  `json:"generate_audio"`
		ServiceTier           string `json:"service_tier"`
		ExecutionExpiresAfter int    `json:"execution_expires_after"`
		Priority              int    `json:"priority"`
		Request               struct {
			Resolution string `json:"resolution"`
			Ratio      string `json:"ratio"`
			Duration   int    `json:"duration"`
		} `json:"request"`
	}
	if err := json.Unmarshal(taskData, &stored); err != nil {
		t.Fatalf("taskData is not JSON: %v", err)
	}
	if stored.ID != "upstream_task_123" || stored.Model != "volcengine/seedance-2-0-pro" || stored.Status != "queued" {
		t.Fatalf("stored base fields = %#v", stored)
	}
	if stored.Resolution != "1080p" || stored.Ratio != "16:9" || stored.Duration != 5 {
		t.Fatalf("stored native fields = %#v", stored)
	}
	if stored.GenerateAudio == nil || !*stored.GenerateAudio {
		t.Fatalf("stored generate_audio = %#v, want true", stored.GenerateAudio)
	}
	if stored.ServiceTier != "default" || stored.ExecutionExpiresAfter != 172800 || stored.Priority != 3 {
		t.Fatalf("stored filters = %#v", stored)
	}
	if stored.Request.Resolution != "1080p" || stored.Request.Ratio != "16:9" || stored.Request.Duration != 5 {
		t.Fatalf("stored request snapshot = %#v", stored.Request)
	}
}

func TestDoResponseKeepsOpenAIVideoBodyByDefault(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance-2-0-pro",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public_123",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser(`{"id":"upstream_task_123"}`),
	}

	if _, _, taskErr := (&TaskAdaptor{}).DoResponse(ctx, resp, info); taskErr != nil {
		t.Fatalf("DoResponse returned task error: %v", taskErr)
	}

	var video dto.OpenAIVideo
	if err := json.Unmarshal(recorder.Body.Bytes(), &video); err != nil {
		t.Fatalf("response is not OpenAIVideo: %v", err)
	}
	if video.ID != "task_public_123" || video.TaskID != "task_public_123" {
		t.Fatalf("OpenAIVideo id/task_id = %q/%q, want public id", video.ID, video.TaskID)
	}
	if video.Model != "seedance-2-0-pro" {
		t.Fatalf("OpenAIVideo model = %q, want seedance-2-0-pro", video.Model)
	}
	if video.CreatedAt <= time.Now().Add(-time.Minute).Unix() {
		t.Fatalf("OpenAIVideo created_at = %d, want recent timestamp", video.CreatedAt)
	}
}

func TestConvertToOpenAIVideoReadsNativeCanonicalTaskData(t *testing.T) {
	t.Parallel()

	task := &model.Task{
		TaskID:    "task_public_123",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: time.Now().Add(-time.Minute).Unix(),
		UpdatedAt: time.Now().Unix(),
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		Data: []byte(`{
			"id":"upstream_task_123",
			"model":"volcengine/seedance-2-0-pro",
			"status":"succeeded",
			"content":{"video_url":"https://cdn.example.com/video.mp4"}
		}`),
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	if err != nil {
		t.Fatalf("ConvertToOpenAIVideo returned error: %v", err)
	}
	var video dto.OpenAIVideo
	if err := json.Unmarshal(body, &video); err != nil {
		t.Fatalf("response is not OpenAIVideo: %v", err)
	}
	if video.ID != "task_public_123" || video.TaskID != "task_public_123" {
		t.Fatalf("OpenAIVideo id/task_id = %q/%q, want public id", video.ID, video.TaskID)
	}
	if video.Metadata["url"] != "https://cdn.example.com/video.mp4" {
		t.Fatalf("OpenAIVideo metadata.url = %#v, want canonical video url", video.Metadata["url"])
	}
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
