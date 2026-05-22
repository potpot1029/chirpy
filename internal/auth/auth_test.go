package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCheckPasswordHash(t *testing.T) {
	password1 := "password123456"
	password2 := "password245678"
	hash1, _ := HashPassword(password1)
	hash2, _ := HashPassword(password2)

	tests := []struct {
		name          string
		password      string
		hash          string
		wantErr       bool
		matchPassword bool
	}{
		{
			name:          "correct password hash",
			password:      password1,
			hash:          hash1,
			wantErr:       false,
			matchPassword: true,
		},
		{
			name:          "incorrect password",
			password:      "incorrectpassword",
			hash:          hash1,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "not matching password and hash",
			password:      password1,
			hash:          hash2,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "empty password",
			password:      "",
			hash:          hash1,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "invalid hash",
			password:      password1,
			hash:          "invalidhash",
			wantErr:       true,
			matchPassword: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := CheckPasswordHash(tt.password, tt.hash)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPasswordHash() wantErr = %v, error = %v", tt.wantErr, err)
			}
			if !tt.wantErr && match != tt.matchPassword {
				t.Errorf("CheckPasswordHash expects %v, got %v", tt.matchPassword, match)
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	validToken, _ := MakeJWT(userID, "secret", time.Hour)

	tests := []struct {
		name        string
		tokenString string
		tokenSecret string
		wantErr     bool
		wantUserID  uuid.UUID
	}{
		{
			name:        "valid token",
			tokenString: validToken,
			tokenSecret: "secret",
			wantErr:     false,
			wantUserID:  userID,
		},
		{
			name:        "invalid token",
			tokenString: "invalid token",
			tokenSecret: "secret",
			wantErr:     true,
			wantUserID:  uuid.Nil,
		},
		{
			name:        "incorrect secret",
			tokenString: validToken,
			tokenSecret: "incorrect secret",
			wantErr:     true,
			wantUserID:  uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, err := ValidateJWT(tt.tokenString, tt.tokenSecret)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() wantErr = %v, error = %v", tt.wantErr, err)
			}
			if !tt.wantErr && gotUserID != tt.wantUserID {
				t.Errorf("ValidateJWT() expects %v, got %v", tt.wantUserID, gotUserID)
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	headers1 := http.Header{}
	headers1.Set("Authorization", "Bearer a-bearer-token")

	headers2 := http.Header{}

	tests := []struct {
		name            string
		headers         http.Header
		wantErr         bool
		wantBearerToken string
	}{
		{
			name:            "correct bearer token",
			headers:         headers1,
			wantErr:         false,
			wantBearerToken: "a-bearer-token",
		},
		{
			name:            "missing Authorization header",
			headers:         headers2,
			wantErr:         true,
			wantBearerToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBearerToken, err := GetBearerToken(tt.headers)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBearerToken() wantErr = %v, error = %v", tt.wantErr, err)
			}
			if !tt.wantErr && gotBearerToken != tt.wantBearerToken {
				t.Errorf("GetBearerToken() expects %v, got %v", tt.wantBearerToken, gotBearerToken)
			}
		})
	}
}
