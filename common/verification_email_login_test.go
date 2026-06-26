package common

import "testing"

func TestEmailLoginPurposeIsIsolatedFromRegistrationPurpose(t *testing.T) {
	email := "login-user@example.com"
	code := "123456"

	RegisterVerificationCodeWithKey(email, code, EmailLoginPurpose)

	if VerifyCodeWithKey(email, code, EmailVerificationPurpose) {
		t.Fatal("邮箱登录验证码不能复用注册邮箱验证用途")
	}
	if !VerifyCodeWithKey(email, code, EmailLoginPurpose) {
		t.Fatal("邮箱登录验证码必须能用邮箱登录用途校验")
	}
}
