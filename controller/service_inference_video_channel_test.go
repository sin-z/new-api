package controller

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestChannelTypeDummyUpperBoundIncludesServiceInferenceVideoWithoutPanic(t *testing.T) {
	t.Parallel()

	if constant.ChannelTypeServiceInferenceVideo >= constant.ChannelTypeDummy {
		t.Fatalf("ChannelTypeServiceInferenceVideo = %d must be below ChannelTypeDummy = %d", constant.ChannelTypeServiceInferenceVideo, constant.ChannelTypeDummy)
	}
	if _, ok := channelId2Models[constant.ChannelTypeServiceInferenceVideo]; ok {
		t.Fatalf("channelId2Models contains service-inference.ai, want task-only channel excluded from ordinary model list")
	}
}

func TestServiceInferenceVideoUnsupportedByOrdinaryChannelTest(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("channel-test.go")
	if err != nil {
		t.Fatalf("read channel-test.go: %v", err)
	}
	if !strings.Contains(string(source), "constant.ChannelTypeServiceInferenceVideo") {
		t.Fatalf("channel-test.go unsupportedTestChannelTypes missing ChannelTypeServiceInferenceVideo")
	}
}

func TestAdminChannelEntriesIncludeServiceInferenceVideo(t *testing.T) {
	t.Parallel()

	checkFileContains(t, "../web/default/src/features/channels/constants.ts",
		"102: 'service-inference.ai'",
		"50, 51, 52, 53, 54, 55, 56, 101, 102",
	)
	checkFileContains(t, "../web/default/src/features/channels/lib/channel-utils.ts",
		"102: 'Doubao', // service-inference.ai",
	)
	checkFileContains(t, "../web/classic/src/constants/channel.constants.js",
		"value: 102",
		"label: 'service-inference.ai'",
	)
	checkFileContains(t, "../web/classic/src/helpers/render.jsx",
		"case 102: // service-inference.ai",
	)
}

func checkFileContains(t *testing.T, path string, fragments ...string) {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(source)
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Fatalf("%s missing fragment %q", path, fragment)
		}
	}
}
