package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEmailLoginTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	model.DB = db

	router := gin.New()
	router.Use(sessions.Sessions("email-login-test", cookie.NewStore([]byte("email-login-secret"))))
	router.GET("/api/user/email_login/code", SendEmailLoginCode)
	router.POST("/api/user/email_login", EmailLogin)
	return router
}

func createEmailLoginUser(t *testing.T, user model.User) model.User {
	t.Helper()
	if user.Username == "" {
		user.Username = "email-user"
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	if user.Password == "" {
		user.Password = "unused-password"
	}
	if user.Group == "" {
		user.Group = "default"
	}
	if user.Status == 0 {
		user.Status = common.UserStatusEnabled
	}
	if user.AffCode == "" {
		user.AffCode = strings.ReplaceAll(t.Name(), "/", "_")
	}
	require.NoError(t, model.DB.Create(&user).Error)
	return user
}

func TestSendEmailLoginCodeRejectsUnknownEmail(t *testing.T) {
	router := setupEmailLoginTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/email_login/code?email=missing@example.com", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "email")
	assert.Contains(t, resp.Body.String(), "success\":false")
}

func TestEmailLoginRejectsInvalidCode(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	createEmailLoginUser(t, model.User{Email: "login@example.com"})

	body := bytes.NewBufferString(`{"email":"login@example.com","code":"654321"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":false")
}

func TestSendEmailLoginCodeRejectsDisabledUser(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	createEmailLoginUser(t, model.User{
		Email:  "disabled@example.com",
		Status: common.UserStatusDisabled,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/email_login/code?email=disabled@example.com", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":false")
	assert.Contains(t, resp.Body.String(), "disabled")
}

func TestEmailLoginRejectsDisabledUser(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	user := createEmailLoginUser(t, model.User{
		Email:  "disabled@example.com",
		Status: common.UserStatusDisabled,
	})
	common.RegisterVerificationCodeWithKey(user.Email, "123456", common.EmailLoginPurpose)

	body := bytes.NewBufferString(`{"email":"disabled@example.com","code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":false")
	assert.Contains(t, resp.Body.String(), "disabled")
}

func TestEmailLoginSetsSessionForEnabledUser(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	user := createEmailLoginUser(t, model.User{
		Email:    "login@example.com",
		Username: "email-user",
	})
	common.RegisterVerificationCodeWithKey(user.Email, "123456", common.EmailLoginPurpose)

	body := bytes.NewBufferString(`{"email":"login@example.com","code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":true")
	assert.Contains(t, resp.Body.String(), `"username":"email-user"`)
	assert.True(t, strings.Contains(resp.Header().Get("Set-Cookie"), "email-login-test"))
	assert.False(t, common.VerifyCodeWithKey(user.Email, "123456", common.EmailLoginPurpose), "登录成功后应删除验证码")
}
