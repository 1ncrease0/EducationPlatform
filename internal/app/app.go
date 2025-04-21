package app

import (
	"SkillForge/internal/app/server"
	"SkillForge/internal/config"
	"SkillForge/internal/delivery/http"
	"SkillForge/internal/service"
	"SkillForge/internal/service/auth"
	"SkillForge/internal/storage/postgres"
	"SkillForge/pkg/logger"
	"os"
	"os/signal"
	"syscall"
)

func Run(cfg *config.Config) {

	log := logger.New(cfg.Env)
	log.Info("Starting with Env: " + cfg.Env)

	pg, err := postgres.NewPostgresPool(cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)
	if err != nil {
		log.FatalErr("error connecting to database", err)
	}
	defer pg.Close()

	tokenRepo := postgres.NewTokensPostgres(pg.Pool)
	userRepo := postgres.NewUserPostgres(pg.Pool)
	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, "//", cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	authUsecase := auth.NewAuthUsecase(log, jwtManager, userRepo, tokenRepo)
	u := service.Collection{AuthService: authUsecase}

	r := http.InitRoutes(log, u)

	srv := server.New(cfg.HTTPServer.Address, cfg.HTTPServer.Timeout, cfg.HTTPServer.IdleTimeout, r)
	srv.Start()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		log.Info("app signal: %s" + s.String())
	case err := <-srv.Notify():
		log.ErrorErr("err", err)
	}
	err = srv.Shutdown()
	if err != nil {
		log.Error("err", err)
	}
}
