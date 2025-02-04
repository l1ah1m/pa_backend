package server

import (
	"github.com/Point-AI/backend/config"
	_ "github.com/Point-AI/backend/docs"
	apiDelivery "github.com/Point-AI/backend/internal/api/delivery"
	messengerDelivery "github.com/Point-AI/backend/internal/messenger/delivery"
	systemDelivery "github.com/Point-AI/backend/internal/system/delivery"
	authDelivery "github.com/Point-AI/backend/internal/user/delivery"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	echoSwagger "github.com/swaggo/echo-swagger"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"sync"
)

// RunHTTPServer
// @title PointAI
// @version 1.0
// @description This is the backend server for PointAI.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name .
// @license.url .
// @host petstore.swagger.io
// @externalDocs.description  OpenAPI 2.0
// @BasePath /
func RunHTTPServer(cfg *config.Config, db *mongo.Database, str *minio.Client) {
	e := echo.New()

	logger := logrus.New()
	logger.Out = os.Stdout

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339_nano} [${status}] ${method} ${uri} (${latency_human})\n",
		Output: logger.Out,
	}))
	e.Use(middleware.CORS())

	repoMu, storageMu := new(sync.RWMutex), new(sync.RWMutex)

	authDelivery.RegisterAuthRoutes(e, cfg, db, str, repoMu, storageMu)
	systemDelivery.RegisterSystemRoutes(e, cfg, db, str, repoMu, storageMu)
	apiDelivery.RegisterAPIRoutes(e, cfg, db)
	messengerDelivery.RegisterMessengerRoutes(e, cfg, db, repoMu)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	if err := e.Start(cfg.Server.Port); err != nil {
		panic(err)
	}
}
