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
	demoMode     bool
}

func NewVideoController(svc *videoservice.VideoService, storagePath, serverURL string, demoMode bool) *VideoController {
	return &VideoController{videoService: svc, storagePath: storagePath, serverURL: serverURL, demoMode: demoMode}
}

func (c *VideoController) ShowUploadPage(ctx *gin.Context) {
	if c.demoMode {
		ctx.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	username, _ := ctx.Get("username")
	ctx.HTML(http.StatusOK, "upload.html", gin.H{
		"title":     "Upload a video — TimmyGram",
		"page":      "upload",
		"serverURL": c.serverURL,
		"username":  username,
		"demoMode":  c.demoMode,
	})
}

func (c *VideoController) HandleUpload(ctx *gin.Context) {
	if c.demoMode {
		ctx.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}
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
			"demoMode":  c.demoMode,
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

func (c *VideoController) UpdateVideo(ctx *gin.Context) {
	if c.demoMode {
		ctx.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	userID := ctx.GetInt("user_id")
	title := ctx.PostForm("title")

	if err := c.videoService.UpdateTitle(id, userID, title); err != nil {
		if errors.Is(err, model.ErrVideoNotFound) {
			ctx.Status(http.StatusNotFound)
		} else if errors.Is(err, model.ErrVideoForbidden) {
			ctx.Status(http.StatusForbidden)
		} else {
			ctx.Status(http.StatusInternalServerError)
		}
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/dashboard")
}

func (c *VideoController) DeleteVideo(ctx *gin.Context) {
	if c.demoMode {
		ctx.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	userID := ctx.GetInt("user_id")
	if err := c.videoService.DeleteVideo(id, userID); err != nil {
		if errors.Is(err, model.ErrVideoNotFound) {
			ctx.Status(http.StatusNotFound)
		} else if errors.Is(err, model.ErrVideoForbidden) {
			ctx.Status(http.StatusForbidden)
		} else {
			ctx.Status(http.StatusInternalServerError)
		}
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/dashboard")
}

func (c *VideoController) LikeVideo(ctx *gin.Context) {
	v, ok := c.getAPIVideo(ctx)
	if !ok {
		return
	}

	deviceID := ctx.GetString("device_id")
	if deviceID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required."})
		return
	}

	likesCount, err := c.videoService.LikeVideo(v.ID, ctx.GetInt("user_id"), deviceID)
	if err != nil {
		if errors.Is(err, model.ErrVideoNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Video not found."})
		} else if errors.Is(err, model.ErrVideoForbidden) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "Access denied."})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like video."})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"video_id":    v.ID,
		"likes_count": likesCount,
	})
}

func (c *VideoController) GetVideoLikes(ctx *gin.Context) {
	v, ok := c.getAPIVideo(ctx)
	if !ok {
		return
	}

	likesCount, err := c.videoService.CountLikes(v.ID, ctx.GetInt("user_id"))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count likes."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"video_id":    v.ID,
		"likes_count": likesCount,
	})
}

func (c *VideoController) VideoStatus(ctx *gin.Context) {
	v, ok := c.getOwnedVideo(ctx)
	if !ok {
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id":            v.ID,
		"title":         v.Title,
		"duration":      v.Duration,
		"thumbnail":     v.Thumbnail,
		"is_processing": v.IsProcessing(),
		"likes_count":   v.LikesCount,
		"transcoded_at": v.TranscodedAt,
		"thumbnail_url": "/videos/" + strconv.Itoa(v.ID) + "/thumbnail",
		"stream_url":    "/videos/" + strconv.Itoa(v.ID) + "/stream",
	})
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
