// utils/email.go
package utils

import (
	"fmt"
	"gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body, from, host, username, password string, port int) error {
	fmt.Println("Sending email to:", to, "with subject:", subject, "and body:", body,
		"from:", from, "host:", host, "username:", username, "port:", port, " password:", password)
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// 注意：使用时需配置正确的SMTP信息和密码
	d := gomail.NewDialer(host, port, username, password)
	return d.DialAndSend(m)
}
