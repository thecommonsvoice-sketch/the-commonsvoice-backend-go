package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/services"
)

type ContactHandler struct {
	Cfg *config.Config
}

func NewContactHandler(cfg *config.Config) *ContactHandler {
	return &ContactHandler{Cfg: cfg}
}

func (h *ContactHandler) SendContactMail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Subject  string `json:"subject"`
		Category string `json:"category"`
		Message  string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || req.Subject == "" || req.Message == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"success": false, "message": "Required fields missing",
		})
		return
	}

	if req.Category == "" {
		req.Category = "General"
	}

	htmlBody := fmt.Sprintf(`
	<div style="font-family: Arial, sans-serif; padding: 20px;">
		<h2>Contact Form Submission</h2>
		<p><strong>Name:</strong> %s</p>
		<p><strong>Email:</strong> %s</p>
		<p><strong>Category:</strong> %s</p>
		<p><strong>Subject:</strong> %s</p>
		<p><strong>Message:</strong></p>
		<p>%s</p>
	</div>
	`, req.Name, req.Email, req.Category, req.Subject, req.Message)

	go func() {
		if err := services.SendMail(
			"smtp.zoho.in", 465,
			h.Cfg.ZohoEmail, h.Cfg.ZohoAppPassword,
			`"The Commons Voice Contact Form" <`+h.Cfg.ZohoEmail+`>`,
			h.Cfg.ZohoEmail,
			req.Email,
			fmt.Sprintf("[%s] %s", req.Category, req.Subject),
			htmlBody,
		); err != nil {
			slog.Error("failed to send contact mail", "error", err, "from", req.Email)
		}
	}()

	response.JSON(w, http.StatusOK, map[string]any{
		"success": true, "message": "Mail sent successfully",
	})
}
