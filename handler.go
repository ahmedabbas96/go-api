package main

import (
	"time"
	"context"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /healthz -> basic "process is alive"
func (a *App) HandleHealthz(c *gin.Context){
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /readyz -> check DB connectivity
func (a *App) HandleReadyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := a.DB.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not-ready", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// POST /userCreate
// { "username": "...", "email": "...", "password": "..." }
func (a *App) HandleUserCreate(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	hash, err := a.HashPassword(body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	id, err := a.CreateUser(body.Username, body.Email, hash)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not create user (maybe username/email already exists)"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user created",
		"user_id": id,
	})
}

// POST /login
// { "username": "...", "password": "..." }
func (a *App) HandleLogin(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	userID, username, passwordHash, err := a.GetUserByUsername(body.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	if err := a.CheckPassword(passwordHash, body.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := a.GenerateJWT(userID, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GET /userDetails (Authorization: Bearer <token>)
func (a *App) HandleUserDetails(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(int)

	user, err := a.GetUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ===== DB helpers =====

func (a *App) CreateUser(username, email, passwordHash string) (int64, error) {
	res, err := a.DB.Exec(`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		username, email, passwordHash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (a *App) GetUserByUsername(username string) (id int, uname, passwordHash string, err error) {
	err = a.DB.QueryRow(`SELECT id, username, password_hash FROM users WHERE username = ?`, username).
		Scan(&id, &uname, &passwordHash)
	return
}

func (a *App) GetUserByID(id int) (*User, error) {
	var u User
	err := a.DB.QueryRow(`SELECT id, username, email, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}