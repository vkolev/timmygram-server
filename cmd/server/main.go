package main

import (
	"timmygram/internal/config"
	authcontroller "timmygram/internal/controller/auth"
	devicecontroller "timmygram/internal/controller/device"
	maincontroller "timmygram/internal/controller/main"
	videocontroller "timmygram/internal/controller/video"
	"timmygram/internal/middleware"
	"timmygram/internal/repository"
	authservice "timmygram/internal/service/auth"
	deviceservice "timmygram/internal/service/device"
	videoservice "timmygram/internal/service/video"
	"timmygram/migrations"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

func createRenderer() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	base := "templates/layouts/base.html"
	dashboard := "templates/layouts/dashboard.html"

	for _, p := range []string{"home", "about", "login", "setup"} {
		r.AddFromFiles(p+".html", "templates/pages/"+p+".html", base)
	}
	for _, p := range []string{"dashboard", "upload", "devices"} {
		r.AddFromFiles(p+".html", "templates/pages/"+p+".html", dashboard)
	}
	return r
}

func main() {
	cfg := config.Load()

	db, err := repository.NewSQLiteDB(cfg.DBPath)
	if err != nil {
		panic(err)
	}

	// Apply incremental migrations idempotently.
	if err := migrations.Run(db, "migrations"); err != nil {
		panic(err)
	}

	userRepo := repository.NewUserRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)

	authSvc := authservice.NewAuthService(userRepo, cfg.JWTSecret)
	videoSvc := videoservice.NewVideoService(videoRepo, cfg.Storage.Path, cfg.FFmpeg.MaxDuration, cfg.FFmpeg.OutputRatio)
	deviceSvc := deviceservice.NewDeviceService(deviceRepo, videoRepo, cfg.JWTSecret, cfg.Server.URL)

	authCtrl := authcontroller.NewAuthController(authSvc, cfg.Server.URL, cfg)
	mainCtrl := maincontroller.NewMainController(cfg.Server.URL, videoSvc)
	videoCtrl := videocontroller.NewVideoController(videoSvc, cfg.Storage.Path, cfg.Server.URL)
	deviceCtrl := devicecontroller.NewDeviceController(deviceSvc, cfg.Server.URL, cfg)

	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MB in memory, rest spills to disk
	router.Static("/static", "./static")
	router.HTMLRender = createRenderer()

	router.Use(middleware.SessionMiddleware(cfg.JWTSecret))
	// Detect if first run and redirect to setup page.
	router.Use(middleware.FirstRunMiddleware(userRepo))

	public := router.Group("/")
	{
		public.GET("", mainCtrl.Index)
		public.GET("/about", mainCtrl.About)
		public.GET("/login", authCtrl.Login)
		public.POST("/login", authCtrl.HandleLogin)
		public.POST("/setup", authCtrl.HandleSetup)
		public.POST("/logout", authCtrl.HandleLogout)
		public.GET("/setup", authCtrl.Setup)
	}

	protected := router.Group("/")
	protected.Use(middleware.RequireSession())
	{
		protected.GET("/dashboard", mainCtrl.Dashboard)
		protected.GET("/upload", videoCtrl.ShowUploadPage)
		protected.POST("/upload", videoCtrl.HandleUpload)
		protected.GET("/videos/:id/stream", videoCtrl.StreamVideo)
		protected.GET("/videos/:id/thumbnail", videoCtrl.ServeThumbnail)
		protected.GET("/videos/:id/status", videoCtrl.VideoStatus)
		protected.POST("/videos/:id", videoCtrl.UpdateVideo)
		protected.POST("/videos/:id/delete", videoCtrl.DeleteVideo)
		protected.GET("/devices", deviceCtrl.ShowDevicesPage)
		protected.GET("/devices/qr", deviceCtrl.ServeQRCode)
		protected.POST("/devices/:id/block", deviceCtrl.BlockDevice)
		protected.POST("/devices/:id/unblock", deviceCtrl.UnblockDevice)
		protected.POST("/devices/:id/delete", deviceCtrl.DeleteDevice)
	}

	api := router.Group("/api/v1")
	api.Use(middleware.DeviceAuthMiddleware(cfg.JWTSecret))
	{
		api.POST("/devices/ping", deviceCtrl.HandlePing)

		activeDeviceAPI := api.Group("/")
		activeDeviceAPI.Use(middleware.RequireActiveDevice(deviceRepo))
		{
			activeDeviceAPI.GET("/feed", deviceCtrl.GetFeed)
			activeDeviceAPI.GET("/feed/:page", deviceCtrl.GetFeed)
			activeDeviceAPI.GET("/next", deviceCtrl.GetNext)
			activeDeviceAPI.GET("/videos/:id/stream", videoCtrl.APIStreamVideo)
			activeDeviceAPI.GET("/videos/:id/thumbnail", videoCtrl.APIServeThumbnail)
		}
	}

	if err := router.Run("0.0.0.0:" + cfg.Server.Port); err != nil {
		return
	}
}
