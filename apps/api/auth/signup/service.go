package signup

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	"github.com/qolzam/telar/apps/api/auth/repository"
	"github.com/qolzam/telar/apps/api/auth/security"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	platformemail "github.com/qolzam/telar/apps/api/internal/platform/email"

	"github.com/qolzam/telar/apps/api/internal/utils"
	"golang.org/x/crypto/bcrypt"
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

type Service struct {
	verificationRepo repository.VerificationRepository
	config           *ServiceConfig
	emailSender      platformemail.Sender // optional; if nil, no email is sent
}

type ServiceConfig struct {
	JWTConfig  platformconfig.JWTConfig
	HMACConfig platformconfig.HMACConfig
	AppConfig  platformconfig.AppConfig
}

func NewService(verificationRepo repository.VerificationRepository, config *ServiceConfig) *Service {
	return &Service{
		verificationRepo: verificationRepo,
		config:           config,
	}
}

// WithEmailSender sets the email sender dependency on the signup service.
func (s *Service) WithEmailSender(sender platformemail.Sender) *Service {
	s.emailSender = sender
	return s
}

func (s *Service) SaveUserVerification(ctx context.Context, userVerification *models.UserVerification) error {
	if err := s.verificationRepo.SaveVerification(ctx, userVerification); err != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to save user verification: %w", err))
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

	// Send email with verification code and link using injected sender when available
	if s.emailSender != nil {
		verificationLink := fmt.Sprintf("%s/verify?verificationId=%s&code=%s", 
			s.config.AppConfig.WebDomain, verifyId.String(), code)
		
		body := fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #1976d2;">Welcome to Telar, %s!</h2>
  <p style="font-size: 16px; color: #333;">Thank you for signing up. Please verify your email address to get started.</p>
  
  <div style="margin: 30px 0;">
    <h3 style="color: #555; font-size: 18px;">Option 1: Click the Button</h3>
    <a href="%s" style="display: inline-block; padding: 14px 28px; background-color: #1976d2; color: white; text-decoration: none; border-radius: 6px; font-weight: bold; font-size: 16px;">Verify Email Address</a>
  </div>
  
  <div style="margin: 30px 0; padding: 20px; background-color: #f5f5f5; border-radius: 8px;">
    <h3 style="color: #555; font-size: 18px; margin-top: 0;">Option 2: Enter Code Manually</h3>
    <p style="color: #666;">If the button doesn't work, enter this code on the verification page:</p>
    <div style="background: white; padding: 16px; text-align: center; font-size: 32px; font-weight: bold; letter-spacing: 8px; border-radius: 4px; color: #1976d2; margin: 10px 0; border: 2px solid #1976d2;">%s</div>
  </div>
  
  <p style="color: #999; font-size: 13px; margin-top: 40px; border-top: 1px solid #eee; padding-top: 20px;">This code expires in 15 minutes.</p>
  <p style="color: #999; font-size: 13px;">If you didn't request this, please ignore this email.</p>
</div>
`, input.FullName, verificationLink, code)
		
		_ = s.emailSender.Send(ctx, platformemail.Message{
			From:    "noreply@telar.dev",
			To:      []string{input.EmailTo},
			Subject: "Verify your Telar account",
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

// ResendVerificationEmail resends verification email with a new code
func (s *Service) ResendVerificationEmail(ctx context.Context, verificationId uuid.UUID) error {
	// Find verification by ID
	verification, err := s.verificationRepo.FindByID(ctx, verificationId)
	if err != nil {
		return errors.WrapUserNotFoundError(fmt.Errorf("verification not found: %w", err))
	}

	if verification.Used || verification.IsVerified {
		return errors.WrapValidationError(fmt.Errorf("verification already completed"), "verificationId")
	}

	newCode := utils.GenerateDigits(6)
	expiresAt := time.Now().Add(15 * time.Minute).Unix()

	// Update verification with new code
	if err := s.verificationRepo.UpdateVerificationCode(ctx, verificationId, newCode, expiresAt); err != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to update verification: %w", err))
	}

	if s.emailSender != nil {
		verificationLink := fmt.Sprintf("%s/verify?verificationId=%s&code=%s",
			s.config.AppConfig.WebDomain, verificationId.String(), newCode)

		fullName := verification.FullName
		if fullName == "" {
			fullName = "User" // Fallback if full name is not available
		}

		body := fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #1976d2;">Hi %s!</h2>
  <p style="font-size: 16px; color: #333;">Here's your new verification code:</p>
  
  <div style="margin: 30px 0;">
    <h3 style="color: #555; font-size: 18px;">Option 1: Click the Button</h3>
    <a href="%s" style="display: inline-block; padding: 14px 28px; background-color: #1976d2; color: white; text-decoration: none; border-radius: 6px; font-weight: bold; font-size: 16px;">Verify Email Address</a>
  </div>
  
  <div style="margin: 30px 0; padding: 20px; background-color: #f5f5f5; border-radius: 8px;">
    <h3 style="color: #555; font-size: 18px; margin-top: 0;">Option 2: Enter Code Manually</h3>
    <div style="background: white; padding: 16px; text-align: center; font-size: 32px; font-weight: bold; letter-spacing: 8px; border-radius: 4px; color: #1976d2; margin: 10px 0; border: 2px solid #1976d2;">%s</div>
  </div>
  
  <p style="color: #999; font-size: 13px; margin-top: 40px;">This code expires in 15 minutes.</p>
</div>
`, fullName, verificationLink, newCode)
		
		_ = s.emailSender.Send(ctx, platformemail.Message{
			From:    "noreply@telar.dev",
			To:      []string{verification.Target},
			Subject: "Your new Telar verification code",
			Body:    body,
		})
	}
	
	return nil
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
// This method is kept for backward compatibility but should be replaced with specific repository methods
func (s *Service) UpdateVerification(ctx context.Context, filter *models.DatabaseFilter, data *models.DatabaseUpdate) error {
	if filter == nil || filter.ObjectId == nil {
		return errors.WrapValidationError(fmt.Errorf("verification ID is required"), "objectId")
	}

	// Verify verification exists
	_, err := s.verificationRepo.FindByID(ctx, *filter.ObjectId)
	if err != nil {
		return errors.WrapUserNotFoundError(fmt.Errorf("verification not found: %w", err))
	}

	// Handle updates from DatabaseUpdate
	if data != nil && data.Set != nil {
		// Update code if provided
		if newCode, ok := data.Set["code"].(string); ok {
			newExpiresAt := time.Now().Add(15 * time.Minute).Unix()
			if expiresAt, ok := data.Set["expiresAt"].(int64); ok {
				newExpiresAt = expiresAt
			}
			if err := s.verificationRepo.UpdateVerificationCode(ctx, *filter.ObjectId, newCode, newExpiresAt); err != nil {
				return errors.WrapDatabaseError(fmt.Errorf("failed to update verification code: %w", err))
			}
		}

		// Mark as verified if requested
		if isVerified, ok := data.Set["isVerified"].(bool); ok && isVerified {
			if err := s.verificationRepo.MarkVerified(ctx, *filter.ObjectId); err != nil {
				return errors.WrapDatabaseError(fmt.Errorf("failed to mark verification as verified: %w", err))
			}
		}

		// Mark as used if requested
		if used, ok := data.Set["used"].(bool); ok && used {
			if err := s.verificationRepo.MarkUsed(ctx, *filter.ObjectId); err != nil {
				return errors.WrapDatabaseError(fmt.Errorf("failed to mark verification as used: %w", err))
			}
		}
	}

	return nil
}
