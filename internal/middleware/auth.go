package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/jwk"
)

type CognitoMiddleware struct {
	userPoolID string
	awsRegion  string
	jwksURL    string
	jwksCache  jwk.Set
}

func NewCognitoMiddleware(userPoolID, awsRegion string) (*CognitoMiddleware, error) {
	if userPoolID == "" || awsRegion == "" {
		return nil, fmt.Errorf("userPoolID and awsRegion are required")
	}

	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", awsRegion, userPoolID)

	// Fetch the JWKS
	keySet, err := jwk.Fetch(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %v", err)
	}

	return &CognitoMiddleware{
		userPoolID: userPoolID,
		awsRegion:  awsRegion,
		jwksURL:    jwksURL,
		jwksCache:  keySet,
	}, nil
}

func (cm *CognitoMiddleware) Validate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		// Remove 'Bearer ' prefix if present
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse the JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// Get the key ID from the token header
			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("kid header not found")
			}

			// Get the key from JWKS
			key, found := cm.jwksCache.LookupKeyID(kid)
			if !found {
				// Refresh JWKS cache if key not found
				keySet, err := jwk.Fetch(cm.jwksURL)
				if err != nil {
					return nil, fmt.Errorf("failed to refresh JWKS: %v", err)
				}
				cm.jwksCache = keySet
				key, found = cm.jwksCache.LookupKeyID(kid)
				if !found {
					return nil, fmt.Errorf("key %v not found in JWKS", kid)
				}
			}

			var rawKey interface{}
			if err := key.Raw(&rawKey); err != nil {
				return nil, fmt.Errorf("failed to get raw key: %v", err)
			}

			return rawKey, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("invalid token: %v", err)})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Add claims to context for later use
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("claims", claims)
		}

		c.Next()
	}
}
