package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenExpiry = time.Hour * 24

func GenerateToken(user User, roles []string, country string, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"name":  user.Username,
		"roles": roles,
		"exp":   time.Now().Add(tokenExpiry).Unix(),
		"iat":   time.Now().Unix(),
	}

	// Include country if available
	if country != "" {
		claims["country"] = country
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("signing token failed: %w", err)
	}
	return signed, nil
}

func GetBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func HasAnyRole(userRoles []string, allowed ...string) bool {
	set := map[string]struct{}{}
	for _, r := range userRoles {
		set[r] = struct{}{}
	}
	for _, a := range allowed {
		if _, ok := set[a]; ok {
			return true
		}
	}
	return false
}

type Claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

func ParseToken(tokenStr string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET not set")
	}
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
