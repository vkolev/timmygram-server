package controller

import (
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

	ctx.HTML(http.StatusOK, "devices.html", gin.H{
		"title":     "Child devices — TimmyGram",
		"page":      "devices",
		"serverURL": c.serverURL,
		"username":  username,
		"devices":   devices,
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

func deviceIDFromRequest(ctx *gin.Context) (string, bool) {
	deviceID := strings.TrimSpace(ctx.GetHeader("X-Device-ID"))
	if deviceID == "" {
		deviceID = strings.TrimSpace(ctx.Query("device_id"))
	}

	return deviceID, deviceID != ""
}

func (c *DeviceController) GetFeed(ctx *gin.Context) {
	userID := ctx.GetInt("user_id")

	deviceID, ok := deviceIDFromRequest(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required."})
		return
	}
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

	videos, hasNextPage, err := c.deviceService.GetFeed(userID, deviceID, page, c.cfg.FeedPageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed."})
		return
	}

	type videoItem struct {
		ID           int    `json:"id"`
		Title        string `json:"title"`
		ThumbnailURL string `json:"thumbnail_url"`
		StreamURL    string `json:"stream_url"`
	}

	items := make([]videoItem, 0, len(videos))
	for _, v := range videos {
		items = append(items, videoItem{
			ID:           v.ID,
			Title:        v.Title,
			ThumbnailURL: fmt.Sprintf("%s/api/v1/videos/%d/thumbnail", c.serverURL, v.ID),
			StreamURL:    fmt.Sprintf("%s/api/v1/videos/%d/stream", c.serverURL, v.ID),
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

	deviceID, ok := deviceIDFromRequest(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required."})
		return
	}

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
	}

	ctx.JSON(http.StatusOK, gin.H{"video": videoItem{
		ID:           video.ID,
		Title:        video.Title,
		ThumbnailURL: fmt.Sprintf("%s/api/v1/videos/%d/thumbnail", c.serverURL, video.ID),
		StreamURL:    fmt.Sprintf("%s/api/v1/videos/%d/stream", c.serverURL, video.ID),
	}})
}
