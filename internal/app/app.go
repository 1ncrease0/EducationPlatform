package app

import (
	"SkillForge/internal/app/server"
	"SkillForge/internal/config"
	"SkillForge/internal/delivery/http"
	"SkillForge/internal/service"
	"SkillForge/internal/service/auth"
	"SkillForge/internal/service/course"
	"SkillForge/internal/service/lesson"
	"SkillForge/internal/storage/elastic"
	"SkillForge/internal/storage/minio_storage"
	"SkillForge/internal/storage/postgres"
	"SkillForge/pkg/logger"
	"context"
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

	es, err := elastic.NewElasticClient(cfg.ES.Password, cfg.ES.Hosts)
	if err != nil {
		log.FatalErr("error connecting to elastic", err)
	}

	minio, err := minio_storage.NewMinioStorage(cfg.Minio.Endpoint, cfg.Minio.AccessKey, cfg.Minio.SecretKey, cfg.Minio.UseSSL)
	if err != nil {
		log.FatalErr("error connecting to minio storage", err)
	}

	logoStorage, err := minio_storage.NewLogoStorage(minio, cfg.Minio.Buckets["course_logos"].Name, cfg.Minio.Buckets["course_logos"].PresignTTL)
	if err != nil {
		log.FatalErr("error connecting to minio storage", err)
	}
	lessonMediaStorage, err := minio_storage.NewLessonStorage(minio, cfg.Minio.Buckets["lesson_media"].Name, cfg.Minio.Buckets["lesson_media"].PresignTTL)
	if err != nil {
		log.FatalErr("error connecting to minio storage", err)
	}
	courseES := elastic.NewCourseSearchRepository(es, elastic.CourseIndex)
	err = courseES.CreateIndexIfNotExist(context.Background())
	if err != nil {
		log.FatalErr("error creating index", err)
	}

	tokenRepo := postgres.NewTokensPostgres(pg.Pool)
	courseRepo := postgres.NewCoursePostgres(pg.Pool)
	userRepo := postgres.NewUserPostgres(pg.Pool)
	lessonRepo := postgres.NewLessonPostgres(pg.Pool)
	enrollmentsRepo := postgres.NewSubscriptionPostgres(pg.Pool)
	ratingRepo := postgres.NewCourseRatingPostgres(pg.Pool)

	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, "//", cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	authService := auth.NewAuthService(log, jwtManager, userRepo, tokenRepo)

	courseService := course.NewCourseService(log, courseRepo, courseES, logoStorage, lessonRepo, userRepo, enrollmentsRepo, ratingRepo)
	lessonService := lesson.NewLessonService(log, lessonRepo, courseRepo, lessonMediaStorage)
	u := service.Collection{AuthService: authService, CourseService: courseService, LessonService: lessonService}

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
