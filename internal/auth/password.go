package auth

import (
	"kraken-auth/internal/logger"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password using bcrypt
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares the plaintext with the stored hash using the specified algo.
// Currently supports bcrypt (your seed uses bcrypt). Extend as needed.
func CheckPassword(plain, algo, hash string) bool {
	logger.Infof("[kraken-auth] checking password with algo=%s", algo)

	switch algo {
	case "bcrypt", "":
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
		if err != nil {
			logger.Warnf("[kraken-auth] bcrypt compare failed: %v", err)
			return false
		}
		logger.Infof("[kraken-auth] bcrypt compare success")
		return true
	default:
		logger.Errorf("[kraken-auth] unsupported password algorithm: %s", algo)
		return false
	}
}
