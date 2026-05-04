package main

import (
	version "timmygram/internal"
	"timmygram/internal/config"
	authcontroller "timmygram/internal/controller/auth"
	devicecontroller "timmygram/internal/controller/device"
	maincontroller "timmygram/internal/controller/main"
	usercontroller "timmygram/internal/controller/user"
	videocontroller "timmygram/internal/controller/video"
	"timmygram/internal/middleware"
	"timmygram/internal/repository"
	authservice "timmygram/internal/service/auth"
	deviceservice "timmygram/internal/service/device"
	userservice "timmygram/internal/service/user"
	videoservice "timmygram/internal/service/video"
	"timmygram/migrations"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

type appRenderer struct {
	render    multitemplate.Renderer
	multiUser bool
}

func (r appRenderer) Instance(name string, data any) render.Render {
	if h, ok := data.(gin.H); ok {
		h["AppVersion"] = version.Version
		h["MultiUser"] = r.multiUser
	}
	return r.render.Instance(name, data)
}

func createRenderer(multiUser bool) render.HTMLRender {
	r := multitemplate.NewRenderer()
	base := "templates/layouts/base.html"
	dashboard := "templates/layouts/dashboard.html"

	for _, p := range []string{"home", "about", "login", "setup"} {
		r.AddFromFiles(p+".html", "templates/pages/"+p+".html", base)
	}
	dashboardPages := []string{"dashboard", "upload", "devices", "change_password"}
	if multiUser {
		dashboardPages = append(dashboardPages, "users")
	}
	for _, p := range dashboardPages {
		r.AddFromFiles(p+".html", "templates/pages/"+p+".html", dashboard)
	}
	return appRenderer{render: r, multiUser: multiUser}
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
	mainCtrl := maincontroller.NewMainController(cfg.Server.URL, videoSvc, cfg.DemoMode, cfg.MultiUser)
	videoCtrl := videocontroller.NewVideoController(videoSvc, cfg.Storage.Path, cfg.Server.URL, cfg.DemoMode)
	deviceCtrl := devicecontroller.NewDeviceController(deviceSvc, cfg.Server.URL, cfg)

	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MB in memory, rest spills to disk
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon/favicon.ico")
	router.HTMLRender = createRenderer(cfg.MultiUser)

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

		protected.GET("/account/password", authCtrl.ChangePassword)
		protected.POST("/account/password", authCtrl.HandleChangePassword)
	}

	if cfg.MultiUser {
		userSvc := userservice.NewUserService(userRepo)
		userCtrl := usercontroller.NewUserController(userSvc)

		ownerOnly := router.Group("/")
		ownerOnly.Use(middleware.RequireSession())
		ownerOnly.Use(middleware.RequireOwner())
		{
			ownerOnly.GET("/users", userCtrl.ShowUsersPage)
			ownerOnly.POST("/users", userCtrl.HandleCreateUser)
			ownerOnly.POST("/users/:id/delete", userCtrl.HandleDeleteUser)
		}
	}

	apiPublic := router.Group("/api/v1")
	{
		apiPublic.POST("/auth/pin", deviceCtrl.HandlePINAuth)
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
			activeDeviceAPI.POST("/videos/:id/likes", videoCtrl.LikeVideo)
			activeDeviceAPI.GET("/videos/:id/likes", videoCtrl.GetVideoLikes)
		}
	}

	if err := router.Run("0.0.0.0:" + cfg.Server.Port); err != nil {
		return
	}
}
