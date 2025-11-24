package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// helper to setup gin + app with mocked DB
func newTestApp(t *testing.T) (*App, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	app := &App{
		DB:        db,
		JWTSecret: []byte("test-secret"),
	}

	// Gin in test mode (no noisy logs)
	gin.SetMode(gin.TestMode)

	return app, mock
}

// helper to build a router for tests
func setupRouter(app *App) *gin.Engine {
	r := gin.Default()

	r.POST("/userCreate", app.HandleUserCreate)
	r.POST("/login", app.HandleLogin)

	auth := r.Group("/")
	auth.Use(app.AuthMiddleware())
	auth.GET("/userDetails", app.HandleUserDetails)

	return r
}

// ---------- TEST: POST /userCreate (success) ----------

func TestHandleUserCreate_Success(t *testing.T) {
	app, mock := newTestApp(t)
	router := setupRouter(app)

	// Expect an INSERT into users
	mock.ExpectExec(`INSERT INTO users \(username, email, password_hash\) VALUES \(\?, \?, \?\)`).
		WithArgs("ahmed", "ahmed@example.com", sqlmock.AnyArg()). // password hash is AnyArg
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"username":"ahmed","email":"ahmed@example.com","password":"secret123"}`

	req := httptest.NewRequest(http.MethodPost, "/userCreate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusCreated, w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// ---------- TEST: POST /login (success) ----------

func TestHandleLogin_Success(t *testing.T) {
	app, mock := newTestApp(t)
	router := setupRouter(app)

	// Prepare a bcrypt hash for "secret123"
	hashed, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Expect a SELECT for login
	rows := sqlmock.NewRows([]string{"id", "username", "password_hash"}).
		AddRow(1, "ahmed", string(hashed))

	mock.ExpectQuery(`SELECT id, username, password_hash FROM users WHERE username = \?`).
		WithArgs("ahmed").
		WillReturnRows(rows)

	body := `{"username":"ahmed","password":"secret123"}`

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("expected non-empty token in response")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// ---------- TEST: GET /userDetails (success) ----------

func TestHandleUserDetails_Success(t *testing.T) {
	app, mock := newTestApp(t)
	router := setupRouter(app)

	// Create a valid JWT token for userID=1
	token, err := app.GenerateJWT(1, "ahmed")
	if err != nil {
		t.Fatalf("failed to generate jwt: %v", err)
	}

	// Expect a SELECT by id for userDetails
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "username", "email", "created_at"}).
		AddRow(1, "ahmed", "ahmed@example.com", now)

	mock.ExpectQuery(`SELECT id, username, email, created_at FROM users WHERE id = \?`).
		WithArgs(1).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/userDetails", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var userResp User
	if err := json.Unmarshal(w.Body.Bytes(), &userResp); err != nil {
		t.Fatalf("failed to parse user response: %v", err)
	}
	if userResp.ID != 1 || userResp.Username != "ahmed" {
		t.Fatalf("unexpected user response: %+v", userResp)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
