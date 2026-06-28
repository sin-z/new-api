package doubao

import (
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
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
