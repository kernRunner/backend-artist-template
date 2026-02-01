package auth

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendVerificationEmail(to string, token string) error {
	from := os.Getenv("SMTP_FROM")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	link := fmt.Sprintf("http://localhost:8080/verify?token=%s", token)
	subject := "Verify Your Account"
	body := fmt.Sprintf("Click the following link to verify your account:\n\n%s", link)

	message := []byte("Subject: " + subject + "\r\n" +
		"From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	err := smtp.SendMail(host+":"+port, auth, from, []string{to}, message)
	if err != nil {
		fmt.Println("❌ SMTP error:", err) // <-- ADD THIS
	}
	return err
}
