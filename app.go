package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type App struct {
	DB	*sql.DB
	JWTSecret []byte
}

func NewApp() *App {
	dbUser := getEnv("DB_USER", "root")
	dbPass := getEnv("DB_PASSWORD", "")
	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnv("DB_PORT", "3306")
	dbName := getEnv("DB_NAME", "demo_api")

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set in environment")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPass, dbHost, dbPort, dbName)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("error opening DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("error pinging DB: %v", err)
	}

	return &App{
		DB:        db,
		JWTSecret: []byte(jwtSecret),
	}
}

func (a *App) Run(addr string) error {
	r := gin.Default()

	// Public endpoints
	r.POST("/userCreate", a.HandleUserCreate)
	r.POST("/login", a.HandleLogin)

	// Protected endpoints
	auth := r.Group("/")
	auth.Use(a.AuthMiddleware())
	auth.GET("/userDetails", a.HandleUserDetails)

	log.Printf("server listening on %s\n", addr)
	return r.Run(addr)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}