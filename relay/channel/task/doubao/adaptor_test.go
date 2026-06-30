package doubao

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestURLKeepsAPIV3Path(t *testing.T) {
	t.Parallel()

	adaptor := &TaskAdaptor{}
	adaptor.Init(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://ark.cn-beijing.volces.com",
		},
	})

	got, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{})
	if err != nil {
		t.Fatalf("BuildRequestURL returned error: %v", err)
	}

	want := "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks"
	if got != want {
		t.Fatalf("BuildRequestURL() = %q, want %q", got, want)
	}
}

func TestFetchTaskKeepsAPIV3Path(t *testing.T) {
	t.Parallel()

	adaptor := &TaskAdaptor{}
	_, err := adaptor.FetchTask("://bad-base", "sk-test", map[string]any{"task_id": "task_123"}, "")
	if err == nil || !strings.Contains(err.Error(), `/api/v3/contents/generations/tasks/task_123`) {
		t.Fatalf("FetchTask error = %v, want malformed URL containing Doubao /api/v3 task path", err)
	}
}

func TestDoResponseCanRenderSeedanceNativeCreateResponse(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("seedance_native_response", true)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:  "seedance-2-0-pro",
		Prompt: "city skyline",
		Metadata: map[string]interface{}{
			"service_tier": "default",
			"duration":     5,
		},
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance-2-0-pro",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "public_task_123",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser(`{"id":"upstream_task_123"}`),
	}

	taskID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "upstream_task_123", taskID)
	var data map[string]interface{}
	require.NoError(t, common.Unmarshal(taskData, &data))
	assert.Equal(t, "public_task_123", data["id"])
	assert.Equal(t, "seedance-2-0-pro", data["model"])
	assert.Equal(t, "queued", data["status"])
	assert.Equal(t, "default", data["service_tier"])
	assert.Equal(t, float64(5), data["duration"])
	assert.NotZero(t, data["created_at"])
	assert.NotZero(t, data["updated_at"])
	assert.Equal(t, map[string]interface{}{"video_url": ""}, data["content"])
	assert.Equal(t, map[string]interface{}{"duration": float64(5), "service_tier": "default"}, data["request"])
	assert.NotContains(t, string(taskData), `"id":"upstream_task_123"`)

	var body map[string]string
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, map[string]string{"id": "public_task_123"}, body)
}

func TestConvertToOpenAIVideoUsesCanonicalSeedanceTaskData(t *testing.T) {
	t.Parallel()

	originTask := &model.Task{
		TaskID:    "public_task_123",
		Platform:  constant.TaskPlatform("54"),
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: 1710000000,
		UpdatedAt: 1710000100,
		Properties: model.Properties{
			OriginModelName: "seedance-2-0-pro",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream_task_123",
		},
		Data: []byte(`{
			"id":"public_task_123",
			"model":"seedance-2-0-pro",
			"status":"succeeded",
			"content":{"video_url":"https://cdn.example.com/video.mp4"},
			"request":{"duration":5,"service_tier":"default"}
		}`),
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(originTask)
	require.NoError(t, err)

	var got dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(body, &got))
	assert.Equal(t, "public_task_123", got.ID)
	assert.Equal(t, "public_task_123", got.TaskID)
	assert.Equal(t, "seedance-2-0-pro", got.Model)
	assert.Equal(t, dto.VideoStatusCompleted, got.Status)
	assert.Equal(t, "https://cdn.example.com/video.mp4", got.Metadata["url"])
	assert.NotContains(t, string(body), "upstream_task_123")
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
