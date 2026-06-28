package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestChannelGetBaseURLUsesXRTokenDefault(t *testing.T) {
	t.Parallel()

	channel := &Channel{Type: constant.ChannelTypeXRTokenArkVideo}
	emptyBaseURL := ""
	channel.BaseURL = &emptyBaseURL

	if got := channel.GetBaseURL(); got != "https://api.xrtoken.net" {
		t.Fatalf("GetBaseURL() = %q, want XRToken default", got)
	}
}
