package api

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/docs"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

const EnvDebugMode = "PC_DEBUG_MODE"

var swaggerInstanceCounter uint64

func StartHttpServerWithUnixSocket(useLogger bool, unixSocket string, project app.IProject) (*http.Server, error) {
	router := getRouter(useLogger, project)
	log.Info().Msgf("start UDS http server listening %s", unixSocket)

	// Check if the unix socket is already in use
	// If it exists but we can't connect, remove it
	_, err := net.Dial("unix", unixSocket)
	if err == nil {
		log.Fatal().Msgf("unix socket %s is already in use", unixSocket)
	}
	os.Remove(unixSocket)

	server := &http.Server{
		Handler: router.Handler(),
	}

	listener, err := net.Listen("unix", unixSocket)
	if err != nil {
		return server, err
	}

	go func() {
		defer listener.Close()
		defer os.Remove(unixSocket)

		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msgf("start UDS http server on %s failed", unixSocket)
		}
	}()

	return server, nil
}

func StartHttpServerWithTCP(useLogger bool, address string, port int, project app.IProject) (*http.Server, error) {
	// Register a dedicated Swagger document for this server instance so the
	// interactive Swagger UI points to the configured address and port without
	// mutating the shared default Swagger metadata used by other servers.
	swaggerInfo := *docs.SwaggerInfo
	swaggerInfo.Host = fmt.Sprintf("%s:%d", address, port)
	swaggerInfo.InfoInstanceName = fmt.Sprintf("%s-%d", docs.SwaggerInfo.InstanceName(), atomic.AddUint64(&swaggerInstanceCounter, 1))
	swag.Register(swaggerInfo.InstanceName(), &swaggerInfo)

	router := getRouter(useLogger, project, ginSwagger.InstanceName(swaggerInfo.InstanceName()))
	endPoint := fmt.Sprintf("%s:%d", address, port)
	log.Info().Msgf("start http server listening %s", endPoint)

	server := &http.Server{
		Addr:    endPoint,
		Handler: router.Handler(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msgf("start http server on %s failed", endPoint)
		}
	}()

	return server, nil
}

func getRouter(useLogger bool, project app.IProject, swaggerOptions ...func(*ginSwagger.Config)) *gin.Engine {
	if os.Getenv(EnvDebugMode) == "" {
		gin.SetMode(gin.ReleaseMode)
		useLogger = false
	}
	return InitRoutes(useLogger, NewPcApi(project), swaggerOptions...)
}
