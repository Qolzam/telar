package signup

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	"github.com/qolzam/telar/apps/api/auth/security"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	platformemail "github.com/qolzam/telar/apps/api/internal/platform/email"

	"github.com/qolzam/telar/apps/api/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

const (
	userVerificationCollectionName = "userVerification"
)

// EmailVerificationRequest represents a request to initiate email verification
type EmailVerificationRequest struct {
	UserId          uuid.UUID
	EmailTo         string
	FullName        string
	UserPassword    string
	RemoteIpAddress string
	UserAgent       string
}

type EmailVerificationResponse struct {
	VerificationId string `json:"verificationId"`
	ExpiresAt      int64  `json:"expiresAt"`
	Message        string `json:"message"`
}

// PhoneVerificationRequest represents a request to initiate phone verification
type PhoneVerificationRequest struct {
	UserId          uuid.UUID
	PhoneNumber     string
	FullName        string
	UserPassword    string
	RemoteIpAddress string
	UserAgent       string
}

// PhoneVerificationResponse represents the response from phone verification initiation
type PhoneVerificationResponse struct {
	VerificationId string `json:"verificationId"`
	ExpiresAt      int64  `json:"expiresAt"`
	Message        string `json:"message"`
}

type Service struct{ 
	base *platform.BaseService
	config *ServiceConfig
	emailSender platformemail.Sender // optional; if nil, no email is sent
}

type ServiceConfig struct {
	JWTConfig  platformconfig.JWTConfig
	HMACConfig platformconfig.HMACConfig
	AppConfig  platformconfig.AppConfig
}

func NewService(base *platform.BaseService, config *ServiceConfig) *Service { 
	return &Service{base: base, config: config} 
}

// WithEmailSender sets the email sender dependency on the signup service.
func (s *Service) WithEmailSender(sender platformemail.Sender) *Service {
	s.emailSender = sender
	return s
}

func (s *Service) SaveUserVerification(ctx context.Context, userVerification *models.UserVerification) error {
	result := <-s.base.Repository.Save(ctx, userVerificationCollectionName, userVerification)
	if result.Error != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to save user verification: %w", result.Error))
	}
	return nil
}

// InitiateEmailVerification creates a secure email verification process
func (s *Service) InitiateEmailVerification(ctx context.Context, input EmailVerificationRequest) (*EmailVerificationResponse, error) {
	// Log signup attempt for security monitoring
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: security.EventTypeSignupAttempt,
		UserID:    input.UserId.String(),
		IPAddress: input.RemoteIpAddress,
		UserAgent: input.UserAgent,
		Success:   false, // Will be updated to true on success
		Details:   fmt.Sprintf("Email signup attempt for %s", input.EmailTo),
	})

	// Generate secure verification ID
	verifyId := uuid.Must(uuid.NewV4())

	// Hash password immediately to prevent plaintext storage
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.UserPassword), bcrypt.DefaultCost)
	if err != nil {
		// Log password hashing failure for security monitoring
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: security.EventTypeSignupFailure,
			UserID:    input.UserId.String(),
			IPAddress: input.RemoteIpAddress,
			UserAgent: input.UserAgent,
			Success:   false,
			ErrorCode: "PASSWORD_HASHING_FAILED",
			Details:   "Failed to hash password during signup",
		})
		return nil, errors.WrapSystemError(fmt.Errorf("password hashing failed: %w", err))
	}

	// Generate secure 6-digit verification code
	code := utils.GenerateDigits(6)

	// Calculate expiration time (15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute).Unix()

	// Store verification record securely
	verification := &models.UserVerification{
		ObjectId:        verifyId,
		UserId:          input.UserId,
		Code:            code,
		Target:          input.EmailTo,
		TargetType:      "email",
		HashedPassword:  hashedPassword,
		ExpiresAt:       expiresAt,
		Used:            false,
		IsVerified:      false,
		CreatedDate:     time.Now().Unix(),
		LastUpdated:     time.Now().Unix(),
		RemoteIpAddress: input.RemoteIpAddress,
		Counter:         1,
		FullName:        input.FullName,
	}

	if err := s.SaveUserVerification(ctx, verification); err != nil {
		// Log database failure for security monitoring
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: security.EventTypeSignupFailure,
			UserID:    input.UserId.String(),
			IPAddress: input.RemoteIpAddress,
			UserAgent: input.UserAgent,
			Success:   false,
			ErrorCode: "DATABASE_ERROR",
			Details:   "Failed to save verification record",
		})
		return nil, err
	}

	// Send email with verification code using injected sender when available
	if s.emailSender != nil {
		body := fmt.Sprintf("<p>Hi %s,</p><p>Your verification code is: <b>%s</b></p>", input.FullName, code)
		_ = s.emailSender.Send(ctx, platformemail.Message{
			From:    "noreply@telar.dev",
			To:      []string{input.EmailTo},
			Subject: "Your Telar verification code",
			Body:    body,
		})
	}

	// Log successful signup initiation for security monitoring
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: security.EventTypeSignupSuccess,
		UserID:    input.UserId.String(),
		IPAddress: input.RemoteIpAddress,
		UserAgent: input.UserAgent,
		Success:   true,
		Details:   fmt.Sprintf("Email verification initiated for %s, verification ID: %s", input.EmailTo, verifyId.String()),
	})

	// Return verification details without exposing sensitive data
	return &EmailVerificationResponse{
		VerificationId: verifyId.String(),
		ExpiresAt:      expiresAt,
		Message:        "Verification code sent to your email",
	}, nil
}

// InitiatePhoneVerification creates a secure phone verification process
func (s *Service) InitiatePhoneVerification(ctx context.Context, input PhoneVerificationRequest) (*PhoneVerificationResponse, error) {
	// Log phone signup attempt for security monitoring
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: security.EventTypeSignupAttempt,
		UserID:    input.UserId.String(),
		IPAddress: input.RemoteIpAddress,
		UserAgent: input.UserAgent,
		Success:   false, // Will be updated to true on success
		Details:   fmt.Sprintf("Phone signup attempt for %s", input.PhoneNumber),
	})

	// Generate secure verification ID
	verifyId := uuid.Must(uuid.NewV4())

	// Hash password immediately to prevent plaintext storage
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.UserPassword), bcrypt.DefaultCost)
	if err != nil {
		// Log password hashing failure for security monitoring
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: security.EventTypeSignupFailure,
			UserID:    input.UserId.String(),
			IPAddress: input.RemoteIpAddress,
			UserAgent: input.UserAgent,
			Success:   false,
			ErrorCode: "PASSWORD_HASHING_FAILED",
			Details:   "Failed to hash password during phone signup",
		})
		return nil, errors.WrapSystemError(fmt.Errorf("password hashing failed: %w", err))
	}

	// Generate secure 6-digit verification code
	code := utils.GenerateDigits(6)

	// Calculate expiration time (15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute).Unix()

	// Store verification record securely
	verification := &models.UserVerification{
		ObjectId:        verifyId,
		UserId:          input.UserId,
		Code:            code,
		Target:          input.PhoneNumber,
		TargetType:      "phone",
		HashedPassword:  hashedPassword,
		ExpiresAt:       expiresAt,
		Used:            false,
		IsVerified:      false,
		CreatedDate:     time.Now().Unix(),
		LastUpdated:     time.Now().Unix(),
		RemoteIpAddress: input.RemoteIpAddress,
		Counter:         1,
	}

	if err := s.SaveUserVerification(ctx, verification); err != nil {
		// Log database failure for security monitoring
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: security.EventTypeSignupFailure,
			UserID:    input.UserId.String(),
			IPAddress: input.RemoteIpAddress,
			UserAgent: input.UserAgent,
			Success:   false,
			ErrorCode: "DATABASE_ERROR",
			Details:   "Failed to save phone verification record",
		})
		return nil, err
	}

	// TODO: Implement SMS sending service for phone verification
	// For now, verification code is stored in database for manual testing

	// Log successful phone signup initiation for security monitoring
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: security.EventTypeSignupSuccess,
		UserID:    input.UserId.String(),
		IPAddress: input.RemoteIpAddress,
		UserAgent: input.UserAgent,
		Success:   true,
		Details:   fmt.Sprintf("Phone verification initiated for %s, verification ID: %s", input.PhoneNumber, verifyId.String()),
	})

	// Return verification details without exposing sensitive data
	return &PhoneVerificationResponse{
		VerificationId: verifyId.String(),
		ExpiresAt:      expiresAt,
		Message:        "Verification code sent to your phone",
	}, nil
}




// UpdateVerification updates a verification record in the database
func (s *Service) UpdateVerification(ctx context.Context, filter *models.DatabaseFilter, data *models.DatabaseUpdate) error {
	result := <-s.base.Repository.Update(ctx, userVerificationCollectionName, filter, data, &interfaces.UpdateOptions{})
	if result.Error != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to update verification: %w", result.Error))
	}
	return nil
}
