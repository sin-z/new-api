package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	taskdoubao "github.com/QuantumNous/new-api/relay/channel/task/doubao"
	taskserviceinferencevideo "github.com/QuantumNous/new-api/relay/channel/task/serviceinferencevideo"
	taskxrtokenarkvideo "github.com/QuantumNous/new-api/relay/channel/task/xrtokenarkvideo"
)

func TestXRTokenArkVideoChannelMetadata(t *testing.T) {
	t.Parallel()

	if constant.ChannelTypeXRTokenArkVideo != 101 {
		t.Fatalf("ChannelTypeXRTokenArkVideo = %d, want 101", constant.ChannelTypeXRTokenArkVideo)
	}
	for channelType := 59; channelType < constant.ChannelTypeXRTokenArkVideo; channelType++ {
		if got := constant.ChannelBaseURLs[channelType]; got != "" {
			t.Fatalf("ChannelBaseURLs[%d] = %q, want reserved empty slot", channelType, got)
		}
	}
	if got := constant.ChannelBaseURLs[constant.ChannelTypeXRTokenArkVideo]; got != "https://api.xrtoken.net" {
		t.Fatalf("ChannelBaseURLs[XRTokenArkVideo] = %q, want https://api.xrtoken.net", got)
	}
	if got := constant.GetChannelTypeName(constant.ChannelTypeXRTokenArkVideo); got != "XRTokenArkVideo" {
		t.Fatalf("GetChannelTypeName(XRTokenArkVideo) = %q, want XRTokenArkVideo", got)
	}
}

func TestGetTaskAdaptorReturnsXRTokenArkVideoAdaptor(t *testing.T) {
	t.Parallel()

	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeXRTokenArkVideo)))
	if _, ok := adaptor.(*taskxrtokenarkvideo.TaskAdaptor); !ok {
		t.Fatalf("GetTaskAdaptor(XRTokenArkVideo) = %T, want *xrtokenarkvideo.TaskAdaptor", adaptor)
	}
}

func TestGetTaskAdaptorKeepsDoubaoVideoAdaptor(t *testing.T) {
	t.Parallel()

	adaptor := GetTaskAdaptor(constant.TaskPlatform("54"))
	if _, ok := adaptor.(*taskdoubao.TaskAdaptor); !ok {
		t.Fatalf("GetTaskAdaptor(DoubaoVideo) = %T, want *doubao.TaskAdaptor", adaptor)
	}
}

func TestServiceInferenceVideoChannelMetadata(t *testing.T) {
	t.Parallel()

	if constant.ChannelTypeServiceInferenceVideo != 102 {
		t.Fatalf("ChannelTypeServiceInferenceVideo = %d, want 102", constant.ChannelTypeServiceInferenceVideo)
	}
	if constant.ChannelTypeDummy != 103 {
		t.Fatalf("ChannelTypeDummy = %d, want 103", constant.ChannelTypeDummy)
	}
	if len(constant.ChannelBaseURLs) <= constant.ChannelTypeDummy {
		t.Fatalf("ChannelBaseURLs len = %d, want index %d available", len(constant.ChannelBaseURLs), constant.ChannelTypeDummy)
	}
	if got := constant.ChannelBaseURLs[constant.ChannelTypeServiceInferenceVideo]; got != "https://model.service-inference.ai" {
		t.Fatalf("ChannelBaseURLs[ServiceInferenceVideo] = %q, want https://model.service-inference.ai", got)
	}
	if got := constant.ChannelBaseURLs[constant.ChannelTypeDummy]; got != "" {
		t.Fatalf("ChannelBaseURLs[ChannelTypeDummy] = %q, want empty dummy slot", got)
	}
	if got := constant.GetChannelTypeName(constant.ChannelTypeServiceInferenceVideo); got != "service-inference.ai" {
		t.Fatalf("GetChannelTypeName(ServiceInferenceVideo) = %q, want service-inference.ai", got)
	}
}

func TestGetTaskAdaptorReturnsServiceInferenceVideoAdaptor(t *testing.T) {
	t.Parallel()

	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeServiceInferenceVideo)))
	if _, ok := adaptor.(*taskserviceinferencevideo.TaskAdaptor); !ok {
		t.Fatalf("GetTaskAdaptor(ServiceInferenceVideo) = %T, want *serviceinferencevideo.TaskAdaptor", adaptor)
	}
}
