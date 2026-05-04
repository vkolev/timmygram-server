package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"timmygram/internal/model"
	videoservice "timmygram/internal/service/video"
)

type MainController struct {
	serverURL    string
	videoService *videoservice.VideoService
	demoMode     bool
	multiUser    bool
}

func NewMainController(serverURL string, videoService *videoservice.VideoService, demoMode, multiUser bool) *MainController {
	return &MainController{serverURL: serverURL, videoService: videoService, demoMode: demoMode, multiUser: multiUser}
}

func (c *MainController) HealthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok", "server_url": c.serverURL})
}

func (c *MainController) Index(ctx *gin.Context) {
	isAuth, _ := ctx.Get("is_authenticated")
	ctx.HTML(http.StatusOK, "home.html", gin.H{
		"title":            "TimmyGram",
		"page":             "home",
		"serverURL":        c.serverURL,
		"is_authenticated": isAuth,
	})
}

func (c *MainController) About(ctx *gin.Context) {
	isAuth, _ := ctx.Get("is_authenticated")
	ctx.HTML(http.StatusOK, "about.html", gin.H{
		"title":            "About TimmyGram",
		"page":             "about",
		"serverURL":        c.serverURL,
		"is_authenticated": isAuth,
	})
}

func (c *MainController) Dashboard(ctx *gin.Context) {
	username, _ := ctx.Get("username")
	userID := ctx.GetInt("user_id")
	isOwner := ctx.GetBool("is_owner")

	var videos []*model.Video
	if c.multiUser {
		if v, err := c.videoService.GetAllVideos(); err == nil {
			videos = v
		}
	} else {
		if v, err := c.videoService.GetUserVideos(userID); err == nil {
			videos = v
		}
	}

	ctx.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":     "Dashboard — TimmyGram",
		"page":      "dashboard",
		"serverURL": c.serverURL,
		"username":  username,
		"user_id":   userID,
		"is_owner":  isOwner,
		"videos":    videos,
		"demoMode":  c.demoMode,
	})
}
