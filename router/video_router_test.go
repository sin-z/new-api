package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetVideoRouterRegistersSeedanceNativeRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	SetVideoRouter(r)

	routes := map[string]bool{}
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	require.True(t, routes[http.MethodPost+" /api/v3/contents/generations/tasks"])
	require.True(t, routes[http.MethodGet+" /api/v3/contents/generations/tasks/:task_id"])
	require.True(t, routes[http.MethodGet+" /api/v3/contents/generations/tasks"])
}

func TestSeedanceNativeGetAuthFailureUsesNativeErrorShell(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetVideoRouter(r)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v3/contents/generations/tasks/task_public_123", nil)

	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "OperationDenied.ServiceNotOpen")
	require.Contains(t, recorder.Body.String(), "error")
}
