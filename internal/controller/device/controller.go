package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	deviceservice "timmygram/internal/service/device"
	"timmygram/internal/utils/qr"
)

type DeviceController struct {
	deviceService *deviceservice.DeviceService
	serverURL     string
}

func NewDeviceController(svc *deviceservice.DeviceService, serverURL string) *DeviceController {
	return &DeviceController{deviceService: svc, serverURL: serverURL}
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

func (c *DeviceController) GetFeed(ctx *gin.Context) {
	userID := ctx.GetInt("user_id")

	videos, err := c.deviceService.GetRandomFeed(userID, 5)
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

	ctx.JSON(http.StatusOK, gin.H{"videos": items})
}

func (c *DeviceController) GetNext(ctx *gin.Context) {
	userID := ctx.GetInt("user_id")
	video, err := c.deviceService.GetRandomVideo(userID)
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
