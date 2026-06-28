package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	taskdoubao "github.com/QuantumNous/new-api/relay/channel/task/doubao"
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
