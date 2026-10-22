package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}

	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	})

	sign, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return sign, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("error parsing JWT token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid subject in token: %w", err)
	}

	return userID, nil
}

func getAuthValue(headers http.Header, scheme string) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	value, found := strings.CutPrefix(authHeader, scheme+" ")
	if !found {
		return "", fmt.Errorf("authorization header is not using scheme %s", scheme)
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("authorization header has no value for scheme %s", scheme)
	}

	return value, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	return getAuthValue(headers, "Bearer")
}

func GetAPIKey(headers http.Header) (string, error) {
	return getAuthValue(headers, "ApiKey")
}

func MakeRefreshToken() string {
	key := make([]byte, 32)
	rand.Read(key)
	return hex.EncodeToString(key)
}
