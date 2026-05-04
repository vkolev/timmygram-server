package controller

import (
	"errors"
	"net/http"
	"strings"
	"timmygram/internal/config"

	"github.com/gin-gonic/gin"

	service "timmygram/internal/service/auth"
)

const sessionCookie = "session"
const sessionMaxAge = 24 * 60 * 60

type AuthController struct {
	authService *service.AuthService
	serverURL   string
	cfg         *config.Config
}

func NewAuthController(authService *service.AuthService, serverURL string, cfg *config.Config) *AuthController {
	return &AuthController{authService: authService, serverURL: serverURL, cfg: cfg}
}

func (c *AuthController) Login(ctx *gin.Context) {
	if isAuth, _ := ctx.Get("is_authenticated"); isAuth == true {
		ctx.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}
	ctx.HTML(http.StatusOK, "login.html", gin.H{
		"title":     "Log in to TimmyGram",
		"page":      "login",
		"serverURL": c.serverURL,
	})
}

func (c *AuthController) HandleLogin(ctx *gin.Context) {
	username := strings.TrimSpace(ctx.PostForm("username"))
	password := ctx.PostForm("password")

	renderErr := func(msg string) {
		ctx.HTML(http.StatusUnprocessableEntity, "login.html", gin.H{
			"title":     "Log in to TimmyGram",
			"page":      "login",
			"serverURL": c.serverURL,
			"error":     msg,
			"username":  username,
		})
	}

	if username == "" || password == "" {
		renderErr("Username and password are required.")
		return
	}

	token, err := c.authService.Login(username, password)
	if err != nil {
		renderErr("Invalid username or password.")
		return
	}

	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(sessionCookie, token, sessionMaxAge, "/", "", false, true)
	ctx.Redirect(http.StatusSeeOther, "/dashboard")
}

func (c *AuthController) Setup(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "setup.html", gin.H{
		"title":     "Setup — TimmyGram",
		"page":      "setup",
		"serverURL": c.serverURL,
		"config": gin.H{
			"db_path":           c.cfg.DBPath,
			"storage_path":      c.cfg.Storage.Path,
			"ffmpeg_duration":   c.cfg.FFmpeg.MaxDuration,
			"ffmpeg_ratio":      c.cfg.FFmpeg.OutputRatio,
			"jwt_secret_status": jwtSecretStatus(c.cfg.JWTSecret),
		},
	})
}

func jwtSecretStatus(secret string) bool {
	return secret != ""
}

func (c *AuthController) HandleSetup(ctx *gin.Context) {
	username := strings.TrimSpace(ctx.PostForm("username"))
	password := ctx.PostForm("password")

	renderErr := func(msg string) {
		ctx.HTML(http.StatusUnprocessableEntity, "setup.html", gin.H{
			"title":     "Setup — TimmyGram",
			"page":      "setup",
			"serverURL": c.serverURL,
			"error":     msg,
			"username":  username,
			"config": gin.H{
				"db_path":           c.cfg.DBPath,
				"storage_path":      c.cfg.Storage.Path,
				"ffmpeg_duration":   c.cfg.FFmpeg.MaxDuration,
				"ffmpeg_ratio":      c.cfg.FFmpeg.OutputRatio,
				"jwt_secret_status": jwtSecretStatus(c.cfg.JWTSecret),
			},
		})
	}

	if username == "" || password == "" {
		renderErr("Username and password are required.")
		return
	}
	if len(password) < 6 {
		renderErr("Password must be at least 6 characters.")
		return
	}

	if _, err := c.authService.Register(username, password, true); err != nil {
		if errors.Is(err, service.ErrUsernameTaken) {
			renderErr("That username is already taken.")
		} else {
			renderErr("Registration failed. Please try again.")
		}
		return
	}

	// Auto-login after successful registration.
	token, err := c.authService.Login(username, password)
	if err != nil {
		ctx.Redirect(http.StatusSeeOther, "/login")
		return
	}

	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(sessionCookie, token, sessionMaxAge, "/", "", false, true)
	ctx.Redirect(http.StatusSeeOther, "/dashboard")
}

func (c *AuthController) ChangePassword(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "change_password.html", gin.H{
		"title":    "Change Password — TimmyGram",
		"page":     "change_password",
		"username": ctx.GetString("username"),
		"is_owner": ctx.GetBool("is_owner"),
	})
}

func (c *AuthController) HandleChangePassword(ctx *gin.Context) {
	if c.cfg.DemoMode {
		ctx.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}
	userID := ctx.GetInt("user_id")
	username := ctx.GetString("username")
	isOwner := ctx.GetBool("is_owner")
	current := ctx.PostForm("current_password")
	newPass := ctx.PostForm("new_password")
	confirm := ctx.PostForm("confirm_password")

	renderErr := func(msg string) {
		ctx.HTML(http.StatusUnprocessableEntity, "change_password.html", gin.H{
			"title":    "Change Password — TimmyGram",
			"page":     "change_password",
			"username": username,
			"is_owner": isOwner,
			"error":    msg,
		})
	}

	if current == "" || newPass == "" || confirm == "" {
		renderErr("All fields are required.")
		return
	}
	if len(newPass) < 6 {
		renderErr("New password must be at least 6 characters.")
		return
	}
	if newPass != confirm {
		renderErr("New passwords do not match.")
		return
	}

	if err := c.authService.ChangePassword(userID, current, newPass); err != nil {
		if errors.Is(err, service.ErrWrongPassword) {
			renderErr("Current password is incorrect.")
		} else {
			renderErr("Failed to change password. Please try again.")
		}
		return
	}

	ctx.HTML(http.StatusOK, "change_password.html", gin.H{
		"title":    "Change Password — TimmyGram",
		"page":     "change_password",
		"username": username,
		"is_owner": isOwner,
		"success":  "Password changed successfully.",
	})
}

func (c *AuthController) HandleLogout(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(sessionCookie, "", -1, "/", "", false, true)
	ctx.Redirect(http.StatusSeeOther, "/")
}
