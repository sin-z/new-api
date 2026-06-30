package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetVideoRouterRegistersSeedanceNativeCreateGetListOnly(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetVideoRouter(engine)

	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	assert.True(t, routes[http.MethodPost+" /api/v3/contents/generations/tasks"])
	assert.True(t, routes[http.MethodGet+" /api/v3/contents/generations/tasks/:id"])
	assert.True(t, routes[http.MethodGet+" /api/v3/contents/generations/tasks"])
	assert.False(t, routes[http.MethodDelete+" /api/v3/contents/generations/tasks/:id"])
}
