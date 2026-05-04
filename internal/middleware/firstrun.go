package middleware

import (
	"net/http"
	"timmygram/internal/repository"

	"github.com/gin-gonic/gin"
)

func FirstRunMiddleware(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		hasUsers, err := userRepo.HasUsers()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to check if database is empty."})
			return
		}

		isSetupPath := ctx.Request.URL.Path == "/setup"

		if !hasUsers && !isSetupPath {
			ctx.Redirect(http.StatusFound, "/setup")
			ctx.Abort()
			return
		}

		if hasUsers && isSetupPath {
			ctx.Redirect(http.StatusSeeOther, "/dashboard")
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
