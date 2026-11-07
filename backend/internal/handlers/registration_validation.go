package handlers

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"go-chat/backend/internal/models"
)

func validateRegistration(req models.RegisterRequest) error {
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Email) == "" || req.Password == "" {
		return errors.New("Username, email and password are required")
	}
	if utf8.RuneCountInString(req.Username) > 50 {
		return errors.New("Username cannot exceed 50 characters")
	}
	if utf8.RuneCountInString(req.Email) > 255 {
		return errors.New("Email cannot exceed 255 characters")
	}
	address, err := mail.ParseAddress(req.Email)
	if err != nil || address.Address != req.Email {
		return errors.New("Enter a valid email address")
	}
	if utf8.RuneCountInString(req.Password) < 8 {
		return errors.New("Password must contain at least 8 characters")
	}
	// bcrypt rejects passwords longer than 72 bytes, including UTF-8 encoding.
	if len(req.Password) > 72 {
		return errors.New("Password cannot exceed 72 bytes")
	}
	return nil
}
