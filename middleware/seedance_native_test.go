package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModelRequestTreatsSeedanceNativeCreateAsVideoSubmit(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v3/contents/generations/tasks", strings.NewReader(`{
		"model":"seedance-2-0-pro",
		"content":[{"type":"text","text":"city skyline"}]
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	modelRequest, shouldSelectChannel, err := getModelRequest(c)
	require.NoError(t, err)

	assert.True(t, shouldSelectChannel)
	assert.Equal(t, "seedance-2-0-pro", modelRequest.Model)
	relayMode, exists := c.Get("relay_mode")
	require.True(t, exists)
	assert.Equal(t, relayconstant.RelayModeVideoSubmit, relayMode)
}
