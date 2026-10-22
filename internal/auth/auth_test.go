package auth

import (
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	nonUUIDSubjectToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		Subject:   "not-a-uuid",
	}).SignedString([]byte("secret"))

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
		{
			name:        "subject is not a uuid",
			tokenString: nonUUIDSubjectToken,
			tokenSecret: "secret",
			wantErr:     true,
			wantUserID:  uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID uuid.UUID
			var err error
			if didPanic(func() { gotUserID, err = ValidateJWT(tt.tokenString, tt.tokenSecret) }) {
				t.Fatalf("ValidateJWT() panicked")
			}

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

	headers3 := http.Header{}
	headers3.Set("Authorization", "Bearer")

	headers4 := http.Header{}
	headers4.Set("Authorization", "garbage")

	headers5 := http.Header{}
	headers5.Set("Authorization", "ApiKey a-bearer-token")

	headers6 := http.Header{}
	headers6.Set("Authorization", "Bearer ")

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
		{
			name:            "Bearer without token",
			headers:         headers3,
			wantErr:         true,
			wantBearerToken: "",
		},
		{
			name:            "no space in header",
			headers:         headers4,
			wantErr:         true,
			wantBearerToken: "",
		},
		{
			name:            "wrong scheme",
			headers:         headers5,
			wantErr:         true,
			wantBearerToken: "",
		},
		{
			name:            "Bearer with trailing space but no token",
			headers:         headers6,
			wantErr:         true,
			wantBearerToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBearerToken string
			var err error
			if didPanic(func() { gotBearerToken, err = GetBearerToken(tt.headers) }) {
				t.Fatalf("GetBearerToken() panicked")
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBearerToken() wantErr = %v, error = %v", tt.wantErr, err)
			}
			if !tt.wantErr && gotBearerToken != tt.wantBearerToken {
				t.Errorf("GetBearerToken() expects %v, got %v", tt.wantBearerToken, gotBearerToken)
			}
		})
	}
}

func TestGetAPIKey(t *testing.T) {
	headers1 := http.Header{}
	headers1.Set("Authorization", "ApiKey an-api-key")

	headers2 := http.Header{}

	headers3 := http.Header{}
	headers3.Set("Authorization", "ApiKey")

	headers4 := http.Header{}
	headers4.Set("Authorization", "Bearer an-api-key")

	tests := []struct {
		name       string
		headers    http.Header
		wantErr    bool
		wantAPIKey string
	}{
		{
			name:       "correct api key",
			headers:    headers1,
			wantErr:    false,
			wantAPIKey: "an-api-key",
		},
		{
			name:       "missing Authorization header",
			headers:    headers2,
			wantErr:    true,
			wantAPIKey: "",
		},
		{
			name:       "ApiKey without key",
			headers:    headers3,
			wantErr:    true,
			wantAPIKey: "",
		},
		{
			name:       "wrong scheme",
			headers:    headers4,
			wantErr:    true,
			wantAPIKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotAPIKey string
			var err error
			if didPanic(func() { gotAPIKey, err = GetAPIKey(tt.headers) }) {
				t.Fatalf("GetAPIKey() panicked")
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAPIKey() wantErr = %v, error = %v", tt.wantErr, err)
			}
			if !tt.wantErr && gotAPIKey != tt.wantAPIKey {
				t.Errorf("GetAPIKey() expects %v, got %v", tt.wantAPIKey, gotAPIKey)
			}
		})
	}
}

func TestMakeRefreshToken(t *testing.T) {
	previousToken := MakeRefreshToken()

	tests := []struct {
		name    string
		wantLen int
	}{
		{
			name:    "32 random bytes as hex",
			wantLen: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken := MakeRefreshToken()

			if len(gotToken) != tt.wantLen {
				t.Errorf("MakeRefreshToken() expects length %v, got %v", tt.wantLen, len(gotToken))
			}
			if _, err := hex.DecodeString(gotToken); err != nil {
				t.Errorf("MakeRefreshToken() expects a hex string, got %q", gotToken)
			}
			if gotToken == previousToken {
				t.Errorf("MakeRefreshToken() expects a different token each call, got %q twice", gotToken)
			}
		})
	}
}

// didPanic reports whether f panicked, so one crashing case fails its subtest
// instead of aborting the whole test run.
func didPanic(f func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	f()
	return false
}
