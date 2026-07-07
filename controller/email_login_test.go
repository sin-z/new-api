package controller

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))
	model.DB = db
	model.LOG_DB = db
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
	})
	startEmailLoginTestSMTP(t)

	router := gin.New()
	router.Use(sessions.Sessions("email-login-test", cookie.NewStore([]byte("email-login-secret"))))
	router.GET("/api/user/email_login/code", SendEmailLoginCode)
	router.POST("/api/user/email_login", EmailLogin)
	return router
}

func startEmailLoginTestSMTP(t *testing.T) {
	t.Helper()

	oldSMTPServer := common.SMTPServer
	oldSMTPPort := common.SMTPPort
	oldSMTPAccount := common.SMTPAccount
	oldSMTPFrom := common.SMTPFrom
	oldSMTPToken := common.SMTPToken
	oldSMTPSSLEnabled := common.SMTPSSLEnabled
	oldSMTPStartTLSEnabled := common.SMTPStartTLSEnabled

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)

	common.SMTPServer = host
	common.SMTPPort = port
	common.SMTPAccount = ""
	common.SMTPFrom = "noreply@example.com"
	common.SMTPToken = ""
	common.SMTPSSLEnabled = false
	common.SMTPStartTLSEnabled = false

	t.Cleanup(func() {
		_ = listener.Close()
		common.SMTPServer = oldSMTPServer
		common.SMTPPort = oldSMTPPort
		common.SMTPAccount = oldSMTPAccount
		common.SMTPFrom = oldSMTPFrom
		common.SMTPToken = oldSMTPToken
		common.SMTPSSLEnabled = oldSMTPSSLEnabled
		common.SMTPStartTLSEnabled = oldSMTPStartTLSEnabled
	})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleEmailLoginTestSMTPConn(conn)
		}
	}()
}

func handleEmailLoginTestSMTPConn(conn net.Conn) {
	defer conn.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	writeEmailLoginTestSMTPLine(rw, "220 localhost ESMTP")

	inData := false
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		if inData {
			if command == "." {
				inData = false
				writeEmailLoginTestSMTPLine(rw, "250 OK")
			}
			continue
		}
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			writeEmailLoginTestSMTPLine(rw, "250 localhost")
		case strings.HasPrefix(command, "MAIL FROM"), strings.HasPrefix(command, "RCPT TO"):
			writeEmailLoginTestSMTPLine(rw, "250 OK")
		case command == "DATA":
			inData = true
			writeEmailLoginTestSMTPLine(rw, "354 End data with <CR><LF>.<CR><LF>")
		case command == "QUIT":
			writeEmailLoginTestSMTPLine(rw, "221 Bye")
			return
		default:
			writeEmailLoginTestSMTPLine(rw, "250 OK")
		}
	}
}

func writeEmailLoginTestSMTPLine(rw *bufio.ReadWriter, line string) {
	_, _ = rw.WriteString(line + "\r\n")
	_ = rw.Flush()
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

func setEmailLoginRegisterEnabled(t *testing.T, enabled bool) {
	t.Helper()
	old := common.RegisterEnabled
	common.RegisterEnabled = enabled
	t.Cleanup(func() {
		common.RegisterEnabled = old
	})
}

func setEmailLoginPasswordRegisterEnabled(t *testing.T, enabled bool) {
	t.Helper()
	old := common.PasswordRegisterEnabled
	common.PasswordRegisterEnabled = enabled
	t.Cleanup(func() {
		common.PasswordRegisterEnabled = old
	})
}

func countEmailLoginUsers(t *testing.T, email string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, model.DB.Unscoped().Model(&model.User{}).
		Where("email = ? OR username = ?", email, email).
		Count(&count).Error)
	return count
}

func maxLengthEmailLoginAddress() string {
	localPart := strings.Repeat("a", 64)
	domain := strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 57) + ".com"
	return localPart + "@" + domain
}

func tooLongEmailLoginAddress() string {
	localPart := strings.Repeat("a", 64)
	domain := strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 58) + ".com"
	return localPart + "@" + domain
}

func TestSendEmailLoginCodeAllowsUnknownEmailWhenRegisterEnabled(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	setEmailLoginRegisterEnabled(t, true)

	email := "missing@example.com"
	req := httptest.NewRequest(http.MethodGet, "/api/user/email_login/code?email="+url.QueryEscape(email), nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":true")
	assert.Zero(t, countEmailLoginUsers(t, email), "发送验证码阶段不应提前创建用户")
}

func TestSendEmailLoginCodeRejectsUnknownEmailWhenRegisterDisabled(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	setEmailLoginRegisterEnabled(t, false)

	email := "missing-disabled@example.com"
	req := httptest.NewRequest(http.MethodGet, "/api/user/email_login/code?email="+url.QueryEscape(email), nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":false")
	assert.Zero(t, countEmailLoginUsers(t, email))
}

func TestEmailLoginCreatesUserForUnknownEmail(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	setEmailLoginRegisterEnabled(t, true)
	setEmailLoginPasswordRegisterEnabled(t, false)

	email := "autocreate@example.com"
	common.RegisterVerificationCodeWithKey(email, "123456", common.EmailLoginPurpose)

	body := bytes.NewBufferString(`{"email":"autocreate@example.com","code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":true")
	assert.Contains(t, resp.Body.String(), `"username":"autocreate@example.com"`)
	assert.Contains(t, resp.Body.String(), `"display_name":"autocreate@example.com"`)
	assert.True(t, strings.Contains(resp.Header().Get("Set-Cookie"), "email-login-test"))
	assert.False(t, common.VerifyCodeWithKey(email, "123456", common.EmailLoginPurpose), "登录成功后应删除验证码")

	var user model.User
	require.NoError(t, model.DB.Where("email = ?", email).First(&user).Error)
	assert.Equal(t, email, user.Username)
	assert.Equal(t, email, user.DisplayName)
	assert.Equal(t, common.RoleCommonUser, user.Role)
	assert.Equal(t, common.UserStatusEnabled, user.Status)
	assert.NotEmpty(t, user.Password)
	assert.NotEqual(t, "123456", user.Password)
}

func TestEmailLoginHTTPServerCreatesUserForUnknownEmail(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	setEmailLoginRegisterEnabled(t, true)

	email := "http-autocreate@example.com"
	common.RegisterVerificationCodeWithKey(email, "123456", common.EmailLoginPurpose)

	resp, err := server.Client().Post(
		server.URL+"/api/user/email_login",
		"application/json",
		bytes.NewBufferString(`{"email":"http-autocreate@example.com","code":"123456"}`),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	body := new(bytes.Buffer)
	_, err = body.ReadFrom(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body.String(), "success\":true")
	assert.Contains(t, body.String(), `"username":"http-autocreate@example.com"`)
	assert.NotEmpty(t, resp.Header.Get("Set-Cookie"))
	assert.Equal(t, int64(1), countEmailLoginUsers(t, email))
}

func TestEmailLoginRejectsUnknownEmailWhenRegisterDisabled(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	setEmailLoginRegisterEnabled(t, false)

	email := "login-disabled@example.com"
	common.RegisterVerificationCodeWithKey(email, "123456", common.EmailLoginPurpose)

	body := bytes.NewBufferString(`{"email":"login-disabled@example.com","code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":false")
	assert.Zero(t, countEmailLoginUsers(t, email))
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
	assert.Equal(t, int64(1), countEmailLoginUsers(t, user.Email), "已存在用户邮箱登录不应创建重复账号")
}

func TestEmailLoginAllowsExistingUserWhenRegisterDisabled(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	user := createEmailLoginUser(t, model.User{
		Email:    "existing-disabled-register@example.com",
		Username: "existing-email-user",
	})
	setEmailLoginRegisterEnabled(t, false)
	common.RegisterVerificationCodeWithKey(user.Email, "123456", common.EmailLoginPurpose)

	body := bytes.NewBufferString(`{"email":"existing-disabled-register@example.com","code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":true")
	assert.Contains(t, resp.Body.String(), `"username":"existing-email-user"`)
	assert.Equal(t, int64(1), countEmailLoginUsers(t, user.Email))
}

func TestEmailLoginRejectsSoftDeletedEmail(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	email := "soft-deleted@example.com"
	user := createEmailLoginUser(t, model.User{
		Email:    email,
		Username: email,
	})
	require.NoError(t, model.DB.Delete(&user).Error)
	setEmailLoginRegisterEnabled(t, true)
	common.RegisterVerificationCodeWithKey(email, "123456", common.EmailLoginPurpose)

	sendReq := httptest.NewRequest(http.MethodGet, "/api/user/email_login/code?email="+url.QueryEscape(email), nil)
	sendResp := httptest.NewRecorder()
	router.ServeHTTP(sendResp, sendReq)
	assert.Equal(t, http.StatusOK, sendResp.Code)
	assert.Contains(t, sendResp.Body.String(), "success\":false")

	body := bytes.NewBufferString(`{"email":"soft-deleted@example.com","code":"123456"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	assert.Equal(t, http.StatusOK, loginResp.Code)
	assert.Contains(t, loginResp.Body.String(), "success\":false")
	assert.Equal(t, int64(1), countEmailLoginUsers(t, email), "软删除占用账号不应被重新创建")
}

func TestEmailLoginRejectsEmailWhoseUsernameIsOccupied(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	email := "occupied@example.com"
	createEmailLoginUser(t, model.User{
		Username: email,
		Email:    "other@example.com",
	})
	setEmailLoginRegisterEnabled(t, true)
	common.RegisterVerificationCodeWithKey(email, "123456", common.EmailLoginPurpose)

	sendReq := httptest.NewRequest(http.MethodGet, "/api/user/email_login/code?email="+url.QueryEscape(email), nil)
	sendResp := httptest.NewRecorder()
	router.ServeHTTP(sendResp, sendReq)
	assert.Equal(t, http.StatusOK, sendResp.Code)
	assert.Contains(t, sendResp.Body.String(), "success\":false")

	body := bytes.NewBufferString(`{"email":"occupied@example.com","code":"123456"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	assert.Equal(t, http.StatusOK, loginResp.Code)
	assert.Contains(t, loginResp.Body.String(), "success\":false")
	assert.Equal(t, int64(1), countEmailLoginUsers(t, email))
}

func TestEmailLoginAllowsMaxLengthEmail(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	setEmailLoginRegisterEnabled(t, true)

	email := maxLengthEmailLoginAddress()
	common.RegisterVerificationCodeWithKey(email, "123456", common.EmailLoginPurpose)

	body := bytes.NewBufferString(`{"email":"` + email + `","code":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/email_login", body)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":true")
	var user model.User
	require.NoError(t, model.DB.Where("email = ?", email).First(&user).Error)
	assert.Equal(t, email, user.Username)
	assert.Equal(t, email, user.DisplayName)
}

func TestSendEmailLoginCodeRejectsTooLongEmail(t *testing.T) {
	router := setupEmailLoginTestRouter(t)
	setEmailLoginRegisterEnabled(t, true)

	email := tooLongEmailLoginAddress()
	req := httptest.NewRequest(http.MethodGet, "/api/user/email_login/code?email="+url.QueryEscape(email), nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "success\":false")
	assert.Zero(t, countEmailLoginUsers(t, email))
}
