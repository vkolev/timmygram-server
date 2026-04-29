package middleware

import (
	"timmygram/internal/repository"

	"github.com/gin-gonic/gin"
)

func FirstRunMiddleware(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		hasUsers, err := userRepo.HasUsers()
		if err != nil {
			ctx.AbortWithStatusJSON(500, gin.H{"error": "Failed to check if database is empty."})
			return
		}
		if !hasUsers {
			if ctx.Request.URL.Path == "/setup" {
				ctx.Next()
			}
			ctx.Redirect(302, "/setup")
		}
		ctx.Next()
	}
}
