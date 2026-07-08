package controller

import (
	"bufio"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type capturedEmailSMTPServer struct {
	listener net.Listener
	messages chan string
}

func startCapturedEmailSMTPServer(t *testing.T) *capturedEmailSMTPServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)

	oldSMTPServer := common.SMTPServer
	oldSMTPPort := common.SMTPPort
	oldSMTPAccount := common.SMTPAccount
	oldSMTPFrom := common.SMTPFrom
	oldSMTPToken := common.SMTPToken
	oldSMTPSSLEnabled := common.SMTPSSLEnabled
	oldSMTPStartTLSEnabled := common.SMTPStartTLSEnabled

	common.SMTPServer = host
	common.SMTPPort = port
	common.SMTPAccount = ""
	common.SMTPFrom = "noreply@example.com"
	common.SMTPToken = ""
	common.SMTPSSLEnabled = false
	common.SMTPStartTLSEnabled = false

	server := &capturedEmailSMTPServer{
		listener: listener,
		messages: make(chan string, 8),
	}
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

	go server.serve()
	return server
}

func (s *capturedEmailSMTPServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *capturedEmailSMTPServer) handleConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	writeEmailTemplateSMTPLine(rw, "220 localhost ESMTP")
	inData := false
	var data strings.Builder
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		if inData {
			if command == "." {
				inData = false
				s.messages <- data.String()
				data.Reset()
				writeEmailTemplateSMTPLine(rw, "250 OK")
				continue
			}
			data.WriteString(line)
			continue
		}
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			writeEmailTemplateSMTPLine(rw, "250 localhost")
		case strings.HasPrefix(command, "MAIL FROM"), strings.HasPrefix(command, "RCPT TO"):
			writeEmailTemplateSMTPLine(rw, "250 OK")
		case command == "DATA":
			inData = true
			writeEmailTemplateSMTPLine(rw, "354 End data with <CR><LF>.<CR><LF>")
		case command == "QUIT":
			writeEmailTemplateSMTPLine(rw, "221 Bye")
			return
		default:
			writeEmailTemplateSMTPLine(rw, "250 OK")
		}
	}
}

func writeEmailTemplateSMTPLine(rw *bufio.ReadWriter, line string) {
	_, _ = rw.WriteString(line + "\r\n")
	_ = rw.Flush()
}

func setupEmailTemplateTestRouter(t *testing.T) (*gin.Engine, *capturedEmailSMTPServer) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))
	model.DB = db
	model.LOG_DB = db

	oldSystemName := common.SystemName
	oldRegisterEnabled := common.RegisterEnabled
	oldRedisEnabled := common.RedisEnabled
	oldServerAddress := system_setting.ServerAddress
	common.SystemName = "ZZ123"
	common.RegisterEnabled = true
	common.RedisEnabled = false
	system_setting.ServerAddress = "https://www.zz123.ai"
	t.Cleanup(func() {
		common.SystemName = oldSystemName
		common.RegisterEnabled = oldRegisterEnabled
		common.RedisEnabled = oldRedisEnabled
		system_setting.ServerAddress = oldServerAddress
	})

	server := startCapturedEmailSMTPServer(t)

	router := gin.New()
	router.GET("/api/user/email_login/code", SendEmailLoginCode)
	router.GET("/api/verification", SendEmailVerification)
	router.GET("/api/user/reset", SendPasswordResetEmail)
	return router, server
}

func readCapturedEmail(t *testing.T, server *capturedEmailSMTPServer) (string, string) {
	t.Helper()
	select {
	case raw := <-server.messages:
		message, err := mail.ReadMessage(strings.NewReader(raw))
		require.NoError(t, err)
		subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
		require.NoError(t, err)
		body, err := io.ReadAll(message.Body)
		require.NoError(t, err)
		return subject, string(body)
	case <-time.After(3 * time.Second):
		t.Fatal("等待测试 SMTP 邮件超时")
		return "", ""
	}
}

func requireSuccessfulGet(t *testing.T, router *gin.Engine, target string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
}

func requireEmailCode(t *testing.T, body string) string {
	t.Helper()
	matches := regexp.MustCompile(`Your verification code is: <strong>([^<]+)</strong>`).FindStringSubmatch(body)
	require.Len(t, matches, 2)
	require.Regexp(t, `^\d{6}$`, matches[1])
	require.NotContains(t, body, "验证码")
	return matches[1]
}

func TestSignInCodeEmailUsesEnglishTemplateAndNumericCode(t *testing.T) {
	router, server := setupEmailTemplateTestRouter(t)

	requireSuccessfulGet(t, router, "/api/user/email_login/code?email=signin@example.com")

	subject, body := readCapturedEmail(t, server)
	require.Equal(t, "ZZ123 sign-in code", subject)
	require.Contains(t, body, "You are signing in to ZZ123.")
	require.Contains(t, body, "This code expires in 10 minutes.")
	requireEmailCode(t, body)
}

func TestEmailVerificationUsesEnglishTemplateAndNumericCode(t *testing.T) {
	router, server := setupEmailTemplateTestRouter(t)

	requireSuccessfulGet(t, router, "/api/verification?email=verify@example.com")

	subject, body := readCapturedEmail(t, server)
	require.Equal(t, "ZZ123 email verification code", subject)
	require.Contains(t, body, "You are verifying your email address for ZZ123.")
	require.Contains(t, body, "This code expires in 10 minutes.")
	requireEmailCode(t, body)
}

func TestPasswordResetEmailUsesEnglishTemplateAndKeepsResetLink(t *testing.T) {
	router, server := setupEmailTemplateTestRouter(t)
	createEmailLoginUser(t, model.User{
		Username: "reset-user",
		Email:    "reset@example.com",
	})

	requireSuccessfulGet(t, router, "/api/user/reset?email=reset@example.com")

	subject, body := readCapturedEmail(t, server)
	require.Equal(t, "ZZ123 password reset", subject)
	require.Contains(t, body, "You requested a password reset for ZZ123.")
	require.Contains(t, body, "Click <a href='https://www.zz123.ai/user/reset?email=reset@example.com&token=")
	require.Contains(t, body, "This reset link expires in 10 minutes.")
	require.NotContains(t, body, "密码")
}
