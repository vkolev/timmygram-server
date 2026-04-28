package controller

import (
	"errors"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"timmygram/internal/model"
	videoservice "timmygram/internal/service/video"
)

type VideoController struct {
	videoService *videoservice.VideoService
	storagePath  string
	serverURL    string
}

func NewVideoController(svc *videoservice.VideoService, storagePath, serverURL string) *VideoController {
	return &VideoController{videoService: svc, storagePath: storagePath, serverURL: serverURL}
}

func (c *VideoController) ShowUploadPage(ctx *gin.Context) {
	username, _ := ctx.Get("username")
	ctx.HTML(http.StatusOK, "upload.html", gin.H{
		"title":     "Upload a video — TimmyGram",
		"page":      "upload",
		"serverURL": c.serverURL,
		"username":  username,
	})
}

func (c *VideoController) HandleUpload(ctx *gin.Context) {
	username, _ := ctx.Get("username")
	userID := ctx.GetInt("user_id")
	title := ctx.PostForm("title")

	renderErr := func(msg string) {
		ctx.HTML(http.StatusUnprocessableEntity, "upload.html", gin.H{
			"title":     "Upload a video — TimmyGram",
			"page":      "upload",
			"serverURL": c.serverURL,
			"username":  username,
			"error":     msg,
			"titleVal":  title,
		})
	}

	fileHeader, err := ctx.FormFile("video")
	if err != nil {
		renderErr("Please select a video file.")
		return
	}

	_, err = c.videoService.Upload(userID, title, fileHeader)
	if err != nil {
		switch {
		case errors.Is(err, videoservice.ErrInvalidFileType):
			renderErr("Invalid file type. Please upload a video file.")
		case errors.Is(err, videoservice.ErrFileTooLarge):
			renderErr("File is too large. Maximum size is 200 MB.")
		default:
			renderErr("Upload failed. Please try again.")
		}
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/dashboard")
}

func (c *VideoController) StreamVideo(ctx *gin.Context) {
	v, ok := c.getOwnedVideo(ctx)
	if !ok {
		return
	}
	if v.IsProcessing() {
		ctx.String(http.StatusAccepted, "Video is still processing. Please try again shortly.")
		return
	}
	path := filepath.Join(c.storagePath, "transcoded", v.Filename+".mp4")
	ctx.File(path)
}

func (c *VideoController) ServeThumbnail(ctx *gin.Context) {
	v, ok := c.getOwnedVideo(ctx)
	if !ok {
		return
	}
	if v.Thumbnail == "" {
		ctx.Status(http.StatusNotFound)
		return
	}
	path := filepath.Join(c.storagePath, "thumbnails", v.Thumbnail)
	ctx.File(path)
}

func (c *VideoController) APIStreamVideo(ctx *gin.Context) {
	v, ok := c.getAPIVideo(ctx)
	if !ok {
		return
	}
	if v.IsProcessing() {
		ctx.JSON(http.StatusAccepted, gin.H{"error": "Video is still processing."})
		return
	}
	path := filepath.Join(c.storagePath, "transcoded", v.Filename+".mp4")
	ctx.File(path)
}

func (c *VideoController) APIServeThumbnail(ctx *gin.Context) {
	v, ok := c.getAPIVideo(ctx)
	if !ok {
		return
	}
	if v.Thumbnail == "" {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Thumbnail not found."})
		return
	}
	path := filepath.Join(c.storagePath, "thumbnails", v.Thumbnail)
	ctx.File(path)
}

func (c *VideoController) getAPIVideo(ctx *gin.Context) (*model.Video, bool) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID."})
		return nil, false
	}
	userID := ctx.GetInt("user_id")

	v, err := c.videoService.GetVideo(id, userID)
	if err != nil {
		if errors.Is(err, model.ErrVideoNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Video not found."})
		} else if errors.Is(err, model.ErrVideoForbidden) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "Access denied."})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error."})
		}
		return nil, false
	}
	return v, true
}

func (c *VideoController) getOwnedVideo(ctx *gin.Context) (*model.Video, bool) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return nil, false
	}
	userID := ctx.GetInt("user_id")

	v, err := c.videoService.GetVideo(id, userID)
	if err != nil {
		if errors.Is(err, model.ErrVideoNotFound) {
			ctx.Status(http.StatusNotFound)
		} else if errors.Is(err, model.ErrVideoForbidden) {
			ctx.Status(http.StatusForbidden)
		} else {
			ctx.Status(http.StatusInternalServerError)
		}
		return nil, false
	}
	return v, true
}
