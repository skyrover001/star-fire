// utils/email.go
package utils

import (
	"gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body, from, host, username, password string, port int) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// 注意：使用时需配置正确的SMTP信息和密码
	d := gomail.NewDialer(host, port, username, password)
	return d.DialAndSend(m)
}
