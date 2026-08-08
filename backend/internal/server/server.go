package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MrityunjayRoy/prod-go-boilerplate/internal/config"
	"github.com/MrityunjayRoy/prod-go-boilerplate/internal/db"
	"github.com/MrityunjayRoy/prod-go-boilerplate/internal/lib/job"
	loggerPkg "github.com/MrityunjayRoy/prod-go-boilerplate/internal/logger"
	"github.com/newrelic/go-agent/v3/integrations/nrredis-v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Server struct {
	Config *config.Config
	Logger *zerolog.Logger
	LoggerService *loggerPkg.LoggerService
	DB *db.Database
	Redis *redis.Client
	httpServer *http.Server
	Job *job.JobService
}

func New(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerPkg.LoggerService) (*Server, error) {
	db, err := db.New(cfg, logger, loggerService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize db: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Address,
	})

	if loggerService != nil && loggerService.GetApplication() != nil {
		redisClient.AddHook(nrredis.NewHook(redisClient.Options()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to connect to Redis, continuing without Redis")
		// Don't fail startup if Redis is unavailable
	}

	jobService := job.NewJobService(logger, cfg)
	jobService.InitHandlers(cfg, logger)

	if err := jobService.Start(); err != nil {
		return nil, err
	}

	server := &Server{
		Config: cfg,
		Logger: logger,
		LoggerService: loggerService,
		DB: db,
		Redis: redisClient,
		Job: jobService,
	}

	return server, nil
}

func (server *Server) SetupHTTPServer (handler http.Handler) {
	server.httpServer = &http.Server{
		Addr: ":" + server.Config.Server.Port,
		Handler: handler,
		ReadTimeout: time.Duration(server.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(server.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout: time.Duration(server.Config.Server.IdleTimeout) * time.Second,
	}
}

func (server *Server) Start() error {
	if server.httpServer == nil {
		return errors.New("HTTP server not initialized")
	}

	server.Logger.Info().
		Str("port", server.Config.Server.Port).
		Str("env", server.Config.Primary.Env).
		Msg("starting server...")
	
	return server.httpServer.ListenAndServe()
}

func (server *Server) Shutdown(ctx context.Context) error {
	if err := server.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to Shutdown http server: %w", err)
	}

	if err := server.DB.Close(); err != nil {
		return fmt.Errorf("failed to close the db connection: %w", err)
	}

	if server.Job != nil {
		server.Job.Stop()
	}

	return nil
}
