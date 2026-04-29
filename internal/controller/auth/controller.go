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

	if _, err := c.authService.Register(username, password); err != nil {
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

func (c *AuthController) HandleLogout(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(sessionCookie, "", -1, "/", "", false, true)
	ctx.Redirect(http.StatusSeeOther, "/")
}
