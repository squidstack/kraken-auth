package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken(t *testing.T) {
	user := User{
		ID:       "user123",
		Username: "testuser",
	}
	roles := []string{"admin", "user"}
	secret := "test-secret-key"

	tests := []struct {
		name    string
		user    User
		roles   []string
		secret  string
		wantErr bool
	}{
		{
			name:    "valid token generation",
			user:    user,
			roles:   roles,
			secret:  secret,
			wantErr: false,
		},
		{
			name:    "empty roles",
			user:    user,
			roles:   []string{},
			secret:  secret,
			wantErr: false,
		},
		{
			name:    "empty secret still generates token",
			user:    user,
			roles:   roles,
			secret:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.user, tt.roles, "", tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("GenerateToken() returned empty token")
			}
			if !tt.wantErr {
				// Verify token has 3 parts (header.payload.signature)
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					t.Errorf("token should have 3 parts, got %d", len(parts))
				}
			}
		})
	}
}

func TestGenerateTokenClaims(t *testing.T) {
	user := User{
		ID:       "user456",
		Username: "claimuser",
	}
	roles := []string{"viewer"}
	secret := "claim-secret"

	token, err := GenerateToken(user, roles, "", secret)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	// Parse and verify claims
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse() failed: %v", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to cast claims to MapClaims")
	}

	if claims["sub"] != user.ID {
		t.Errorf("claims[sub] = %v, want %v", claims["sub"], user.ID)
	}
	if claims["name"] != user.Username {
		t.Errorf("claims[name] = %v, want %v", claims["name"], user.Username)
	}

	claimRoles, ok := claims["roles"].([]interface{})
	if !ok || len(claimRoles) != len(roles) {
		t.Errorf("claims[roles] = %v, want %v", claims["roles"], roles)
	}
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "valid bearer token",
			header:   "Bearer abc123",
			expected: "abc123",
		},
		{
			name:     "bearer with lowercase",
			header:   "bearer xyz789",
			expected: "xyz789",
		},
		{
			name:     "no authorization header",
			header:   "",
			expected: "",
		},
		{
			name:     "invalid format no space",
			header:   "Bearertoken",
			expected: "",
		},
		{
			name:     "not bearer scheme",
			header:   "Basic abc123",
			expected: "",
		},
		{
			name:     "empty token",
			header:   "Bearer ",
			expected: "",
		},
		{
			name:     "token with spaces",
			header:   "Bearer token with spaces",
			expected: "token with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			result := GetBearerToken(req)
			if result != tt.expected {
				t.Errorf("GetBearerToken() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHasAnyRole(t *testing.T) {
	tests := []struct {
		name      string
		userRoles []string
		allowed   []string
		expected  bool
	}{
		{
			name:      "has matching role",
			userRoles: []string{"admin", "user"},
			allowed:   []string{"admin"},
			expected:  true,
		},
		{
			name:      "has one of multiple allowed",
			userRoles: []string{"viewer", "editor"},
			allowed:   []string{"admin", "editor", "owner"},
			expected:  true,
		},
		{
			name:      "no matching role",
			userRoles: []string{"guest"},
			allowed:   []string{"admin", "user"},
			expected:  false,
		},
		{
			name:      "empty user roles",
			userRoles: []string{},
			allowed:   []string{"admin"},
			expected:  false,
		},
		{
			name:      "empty allowed roles",
			userRoles: []string{"admin"},
			allowed:   []string{},
			expected:  false,
		},
		{
			name:      "both empty",
			userRoles: []string{},
			allowed:   []string{},
			expected:  false,
		},
		{
			name:      "case sensitive match",
			userRoles: []string{"Admin"},
			allowed:   []string{"admin"},
			expected:  false,
		},
		{
			name:      "exact case match",
			userRoles: []string{"Admin"},
			allowed:   []string{"Admin"},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAnyRole(tt.userRoles, tt.allowed...)
			if result != tt.expected {
				t.Errorf("HasAnyRole(%v, %v) = %v, want %v",
					tt.userRoles, tt.allowed, result, tt.expected)
			}
		})
	}
}

func TestParseToken(t *testing.T) {
	secret := "parse-test-secret"
	os.Setenv("JWT_SECRET", secret)
	defer os.Unsetenv("JWT_SECRET")

	t.Run("valid token", func(t *testing.T) {
		user := User{ID: "user789", Username: "parseuser"}
		roles := []string{"tester"}
		token, err := GenerateToken(user, roles, "", secret)
		if err != nil {
			t.Fatalf("GenerateToken() failed: %v", err)
		}

		claims, err := ParseToken(token)
		if err != nil {
			t.Errorf("ParseToken() error = %v, want nil", err)
		}
		if claims == nil {
			t.Fatal("ParseToken() returned nil claims")
		}
		if claims.Subject != user.ID {
			t.Errorf("claims.Subject = %v, want %v", claims.Subject, user.ID)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := ParseToken("invalid.token.here")
		if err == nil {
			t.Error("ParseToken() should fail for invalid token")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Create token that's already expired
		claims := jwt.MapClaims{
			"sub":   "user999",
			"name":  "expireduser",
			"roles": []string{"viewer"},
			"exp":   time.Now().Add(-time.Hour).Unix(),
			"iat":   time.Now().Add(-2 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, _ := token.SignedString([]byte(secret))

		_, err := ParseToken(signed)
		if err == nil {
			t.Error("ParseToken() should fail for expired token")
		}
	})

	t.Run("missing JWT_SECRET", func(t *testing.T) {
		os.Unsetenv("JWT_SECRET")
		_, err := ParseToken("any.token.here")
		if err == nil {
			t.Error("ParseToken() should fail when JWT_SECRET not set")
		}
		os.Setenv("JWT_SECRET", secret)
	})

	t.Run("wrong secret", func(t *testing.T) {
		user := User{ID: "user999", Username: "wrongsecret"}
		token, _ := GenerateToken(user, []string{"admin"}, "", "wrong-secret")

		_, err := ParseToken(token)
		if err == nil {
			t.Error("ParseToken() should fail with wrong secret")
		}
	})
}

func TestTokenExpiry(t *testing.T) {
	user := User{ID: "expiry-test", Username: "expiryuser"}
	roles := []string{"temp"}
	secret := "expiry-secret"

	token, err := GenerateToken(user, roles, "", secret)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	// Parse token and check expiry is in the future
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse() failed: %v", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to cast claims")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim not found or wrong type")
	}

	expTime := time.Unix(int64(exp), 0)
	if !expTime.After(time.Now()) {
		t.Error("token expiry should be in the future")
	}

	// Check it expires within 25 hours (allowing for some variance)
	if expTime.Sub(time.Now()) > 25*time.Hour {
		t.Error("token expiry is too far in the future")
	}
}
