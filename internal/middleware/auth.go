package middleware

import (
	"errors"
	"net/http"
	"strings"
	"timmygram/internal/model"
	"timmygram/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const sessionCookie = "session"

// SessionMiddleware is a soft auth middleware for browser sessions.
// It reads the session cookie, validates the JWT, and sets is_authenticated,
// user_id, and username in the context. Always calls Next().
func SessionMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString, err := ctx.Cookie(sessionCookie)
		if err != nil || tokenString == "" {
			ctx.Set("is_authenticated", false)
			ctx.Next()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			ctx.SetSameSite(http.SameSiteLaxMode)
			ctx.SetCookie(sessionCookie, "", -1, "/", "", false, true)
			ctx.Set("is_authenticated", false)
			ctx.Next()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			ctx.Set("is_authenticated", false)
			ctx.Next()
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			ctx.Set("is_authenticated", false)
			ctx.Next()
			return
		}

		username, _ := claims["username"].(string)
		ctx.Set("is_authenticated", true)
		ctx.Set("user_id", int(userID))
		ctx.Set("username", username)
		ctx.Next()
	}
}

// RequireSession rejects unauthenticated browser requests with a redirect to /login.
// Must be used after SessionMiddleware.
func RequireSession() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		isAuth, _ := ctx.Get("is_authenticated")
		if auth, ok := isAuth.(bool); !ok || !auth {
			ctx.Redirect(http.StatusSeeOther, "/login")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// DeviceAuthMiddleware accepts only Bearer tokens with type=="device".
func DeviceAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required."})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format."})
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token."})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims."})
			return
		}

		tokenType, _ := claims["type"].(string)
		if tokenType != "device" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token type."})
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token."})
			return
		}

		ctx.Set("user_id", int(userID))
		ctx.Next()
	}
}

func RequireActiveDevice(deviceRepo repository.DeviceRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.GetInt("user_id")

		deviceID := strings.TrimSpace(ctx.GetHeader("X-Device-ID"))
		if deviceID == "" {
			deviceID = strings.TrimSpace(ctx.Query("device_id"))
		}

		if deviceID == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "device_id is required."})
			return
		}

		device, err := deviceRepo.FindByDeviceID(userID, deviceID)
		if err != nil {
			if errors.Is(err, model.ErrDeviceNotFound) {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Device is not registered or has been removed."})
				return
			}

			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify device."})
			return
		}

		if device.IsBlocked() {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Device is blocked."})
			return
		}

		ctx.Set("device_id", device.DeviceID)
		ctx.Set("device", device)
		ctx.Next()
	}
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required."})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format."})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, gin.Error{Err: errors.New("Invalid signing method"), Meta: "Unauthorized"}
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token."})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims."})
			return
		}
		userID, ok := claims["user_id"].(float64)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token claims."})
			return
		}

		ctx.Set("user_id", int(userID))
		ctx.Next()
	}
}
