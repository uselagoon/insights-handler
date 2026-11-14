package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestOneMinuteAdminToken(t *testing.T) {
	for _, audience := range []string{"api.dev", ""} {
		t.Run("audience="+audience, func(t *testing.T) {
			secret := "test-signing-key"
			before := time.Now().Unix()
			tokenString, err := OneMinuteAdminToken(secret, audience, "test-subject", "test-issuer")
			after := time.Now().Unix()
			if err != nil {
				t.Fatal(err)
			}

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil {
				t.Fatal(err)
			}
			if !token.Valid {
				t.Fatal("expected a valid HS256 token")
			}

			for name, want := range map[string]string{
				"role": "admin",
				"sub":  "test-subject",
				"iss":  "test-issuer",
			} {
				if claims[name] != want {
					t.Errorf("claim %s = %v, want %q", name, claims[name], want)
				}
			}
			if audience == "" {
				if _, exists := claims["aud"]; exists {
					t.Error("expected an empty audience to be omitted")
				}
			} else if claims["aud"] != audience {
				t.Errorf("audience = %v, want string %q", claims["aud"], audience)
			}

			issuedAt, ok := claims["iat"].(float64)
			if !ok || issuedAt < float64(before) || issuedAt > float64(after) {
				t.Fatalf("issued-at = %v, want a timestamp between %d and %d", claims["iat"], before, after)
			}
			if claims["exp"] != issuedAt+60 {
				t.Errorf("expiration = %v, want %v", claims["exp"], issuedAt+60)
			}

			if _, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte("incorrect-test-key"), nil
			}, jwt.WithValidMethods([]string{"HS256"})); err == nil {
				t.Error("expected verification with an incorrect key to fail")
			}
		})
	}
}
