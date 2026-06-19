package validator

import "regexp"

var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

// Phone 校验手机号
func Phone(phone string) bool {
	return phoneRegex.MatchString(phone)
}

// Code 校验验证码 (6位纯数字)
func Code(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Nickname 校验昵称 (1-20字符)
func Nickname(name string) bool {
	runes := []rune(name)
	return len(runes) >= 1 && len(runes) <= 20
}
