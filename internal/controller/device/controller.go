package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"timmygram/internal/config"

	"github.com/gin-gonic/gin"

	deviceservice "timmygram/internal/service/device"
	"timmygram/internal/utils/qr"
)

type DeviceController struct {
	deviceService *deviceservice.DeviceService
	serverURL     string
	cfg           *config.Config
}

func NewDeviceController(svc *deviceservice.DeviceService, serverURL string, cfg *config.Config) *DeviceController {
	return &DeviceController{deviceService: svc, serverURL: serverURL, cfg: cfg}
}

func (c *DeviceController) ShowDevicesPage(ctx *gin.Context) {
	username, _ := ctx.Get("username")
	userID := ctx.GetInt("user_id")

	devices, err := c.deviceService.GetDevices(userID)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	pairingPIN, err := c.deviceService.GetOrCreatePairingPIN(userID)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	isOwner := ctx.GetBool("is_owner")
	ctx.HTML(http.StatusOK, "devices.html", gin.H{
		"title":      "Child devices — TimmyGram",
		"page":       "devices",
		"serverURL":  c.serverURL,
		"username":   username,
		"is_owner":   isOwner,
		"devices":    devices,
		"pin":        pairingPIN.PIN,
		"pinExpires": pairingPIN.ExpiresAt,
	})
}

func (c *DeviceController) ServeQRCode(ctx *gin.Context) {
	userID := ctx.GetInt("user_id")

	payload, err := c.deviceService.GenerateQRPayload(userID)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	pngBytes, err := qr.GenerateQRCode(payload)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Data(http.StatusOK, "image/png", pngBytes)
}

type pinAuthRequest struct {
	PIN string `json:"pin" binding:"required"`
}

func (c *DeviceController) HandlePINAuth(ctx *gin.Context) {
	var req pinAuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "PIN is required."})
		return
	}

	token, err := c.deviceService.ExchangePIN(strings.TrimSpace(req.PIN))
	if err != nil {
		if errors.Is(err, deviceservice.ErrInvalidPIN) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired PIN."})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange PIN."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"token": token})
}

func (c *DeviceController) BlockDevice(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	userID := ctx.GetInt("user_id")
	if err := c.deviceService.BlockDevice(id, userID); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/devices")
}

func (c *DeviceController) UnblockDevice(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	userID := ctx.GetInt("user_id")
	if err := c.deviceService.UnblockDevice(id, userID); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/devices")
}

func (c *DeviceController) DeleteDevice(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	userID := ctx.GetInt("user_id")
	if err := c.deviceService.DeleteDevice(id, userID); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/devices")
}

type pingRequest struct {
	DeviceID          string `json:"device_id" binding:"required"`
	DeviceName        string `json:"device_name" binding:"required"`
	DeviceOS          string `json:"device_os"`
	DeviceDescription string `json:"device_description"`
}

func (c *DeviceController) HandlePing(ctx *gin.Context) {
	var req pingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "device_id and device_name are required."})
		return
	}

	userID := ctx.GetInt("user_id")
	if err := c.deviceService.RegisterDevice(userID, req.DeviceID, req.DeviceName); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (c *DeviceController) GetFeed(ctx *gin.Context) {
	userID := ctx.GetInt("user_id")

	page := 1
	pageParam := strings.TrimSpace(ctx.Param("page"))
	if pageParam != "" {
		parsedPage, err := strconv.Atoi(pageParam)
		if err != nil || parsedPage < 1 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number."})
			return
		}
		page = parsedPage
	}

	videos, hasNextPage, err := c.deviceService.GetFeed(userID, page, c.cfg.FeedPageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed."})
		return
	}

	type videoItem struct {
		ID           int    `json:"id"`
		Title        string `json:"title"`
		ThumbnailURL string `json:"thumbnail_url"`
		StreamURL    string `json:"stream_url"`
		LikesCount   int    `json:"likes_count"`
	}

	items := make([]videoItem, 0, len(videos))
	for _, v := range videos {
		items = append(items, videoItem{
			ID:           v.ID,
			Title:        v.Title,
			ThumbnailURL: fmt.Sprintf("%s/api/v1/videos/%d/thumbnail", c.serverURL, v.ID),
			StreamURL:    fmt.Sprintf("%s/api/v1/videos/%d/stream", c.serverURL, v.ID),
			LikesCount:   v.LikesCount,
		})
	}

	var nextPage *string
	if hasNextPage {
		nextPage = new(fmt.Sprintf("%s/api/v1/feed/%d", c.serverURL, page+1))
	}
	ctx.JSON(http.StatusOK, gin.H{"videos": items, "page": page, "next_page": nextPage})
}

func (c *DeviceController) GetNext(ctx *gin.Context) {
	userID := ctx.GetInt("user_id")
	deviceID := ctx.GetString("device_id")

	video, err := c.deviceService.GetRandomVideo(userID, deviceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch video."})
		return
	}

	type videoItem struct {
		ID           int    `json:"id"`
		Title        string `json:"title"`
		ThumbnailURL string `json:"thumbnail_url"`
		StreamURL    string `json:"stream_url"`
		LikesCount   int    `json:"likes_count"`
	}

	ctx.JSON(http.StatusOK, gin.H{"video": videoItem{
		ID:           video.ID,
		Title:        video.Title,
		ThumbnailURL: fmt.Sprintf("%s/api/v1/videos/%d/thumbnail", c.serverURL, video.ID),
		StreamURL:    fmt.Sprintf("%s/api/v1/videos/%d/stream", c.serverURL, video.ID),
		LikesCount:   video.LikesCount,
	}})
}
