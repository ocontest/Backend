package main

import (
	"context"
	"fmt"
	"log"
	"ocontest/api"
	"ocontest/internal/db/mongodb"
	"ocontest/internal/db/postgres"
	"ocontest/internal/jwt"
	"ocontest/internal/minio"
	"ocontest/internal/oc/auth"
	"ocontest/internal/oc/problems"
	"ocontest/internal/otp"
	"ocontest/pkg"

	"github.com/gin-gonic/gin"

	"ocontest/pkg/aes"
	"ocontest/pkg/configs"
	"ocontest/pkg/smtp"
)

func main() {
	configs.InitConf()
	c := configs.Conf
	pkg.InitLog(c.Log)
	pkg.Log.Info("config and log modules initialized")

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	ctx := context.Background()
	// connecting to dependencies
	jwtHandler := jwt.NewGenerator(c.JWT)

	smtpHandler := smtp.NewSMTPHandler(c.SMTP.From, c.SMTP.Password)

	aesHandler, err := aes.NewAesHandler([]byte(c.AESKey))
	if err != nil {
		log.Fatal("error on creating aes handler", err)
	}

	otpStorage := otp.NewOTPStorage()

	dbConn, err := postgres.NewConnectionPool(ctx, c.Postgres)
	if err != nil {
		log.Fatal("error on connecting to db", err)
	}

	minioClient, err := minio.GetNewClient(ctx, c.MinIO)
	if err != nil {
		log.Fatal("error on getting new minio client", err)
	}

	// make repo
	authRepo, err := postgres.NewAuthRepo(ctx, dbConn)
	if err != nil {
		log.Fatal("error on creating auth repo: ", err)
	}
	problemsMetadataRepo, err := postgres.NewProblemsMetadataRepo(ctx, dbConn)
	if err != nil {
		log.Fatal("error on creating problems metadata repo: ", err)
	}

	problemsDescriptionRepo, err := mongodb.NewProblemDescriptionRepo(c.Mongo)
	if err != nil {
		log.Fatal("error on creating problem description repo: ", err)
	}
	// initiating module handlers
	authHandler := auth.NewAuthHandler(authRepo, jwtHandler, smtpHandler, c, aesHandler, otpStorage)
	problemsHandler := problems.NewProblemsHandler(problemsMetadataRepo, problemsDescriptionRepo)
	submissionsHandler := minio.NewSubmissionsHandler(ctx, c.MinIO, minioClient)

	// starting http server
	api.AddRoutes(r, authHandler, problemsHandler, submissionsHandler)

	addr := fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
	pkg.Log.Info("Running on address: ", addr)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
