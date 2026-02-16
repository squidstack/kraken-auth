package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCheckPassword(t *testing.T) {
	// Pre-generate a bcrypt hash for "testpassword"
	validHash, err := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate test hash: %v", err)
	}

	tests := []struct {
		name     string
		plain    string
		algo     string
		hash     string
		expected bool
	}{
		{
			name:     "valid bcrypt password",
			plain:    "testpassword",
			algo:     "bcrypt",
			hash:     string(validHash),
			expected: true,
		},
		{
			name:     "valid with empty algo defaults to bcrypt",
			plain:    "testpassword",
			algo:     "",
			hash:     string(validHash),
			expected: true,
		},
		{
			name:     "invalid password",
			plain:    "wrongpassword",
			algo:     "bcrypt",
			hash:     string(validHash),
			expected: false,
		},
		{
			name:     "unsupported algorithm",
			plain:    "testpassword",
			algo:     "sha256",
			hash:     "somehash",
			expected: false,
		},
		{
			name:     "invalid hash format",
			plain:    "testpassword",
			algo:     "bcrypt",
			hash:     "not-a-valid-hash",
			expected: false,
		},
		{
			name:     "empty password",
			plain:    "",
			algo:     "bcrypt",
			hash:     string(validHash),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPassword(tt.plain, tt.algo, tt.hash)
			if result != tt.expected {
				t.Errorf("CheckPassword(%q, %q, hash) = %v, want %v",
					tt.plain, tt.algo, result, tt.expected)
			}
		})
	}
}

func TestCheckPasswordConcurrent(t *testing.T) {
	// Test thread safety of CheckPassword
	validHash, err := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate test hash: %v", err)
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			result := CheckPassword("testpassword", "bcrypt", string(validHash))
			if !result {
				t.Error("concurrent CheckPassword should succeed")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
