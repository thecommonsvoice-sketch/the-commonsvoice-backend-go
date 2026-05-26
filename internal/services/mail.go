package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
)

func SendMail(host string, port int, username, password, from, to string, replyTo string, subject, body string) error {
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n",
		from, to, subject)
	if replyTo != "" {
		headers += fmt.Sprintf("Reply-To: %s\r\n", replyTo)
	}
	headers += "\r\n"

	msg := []byte(headers + body)

	tlsConfig := &tls.Config{ServerName: host}

	conn, err := tls.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to auth: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set from: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data: %w", err)
	}

	return client.Quit()
}
