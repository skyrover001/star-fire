// utils/email.go
package utils

import (
	"gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "tianyang200202@sina.com")
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// 注意：使用时需配置正确的SMTP信息和密码
	d := gomail.NewDialer("smtp.sina.com", 587, "tianyang200202@sina.com", "c351c82a92b39df8")
	return d.DialAndSend(m)
}
