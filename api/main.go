package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var db *sql.DB

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func connectDB(dsn string) *sql.DB {
	const (
		maxRetries = 20
		retryDelay = 3 * time.Second
	)

	var database *sql.DB
	var err error

	for i := 1; i <= maxRetries; i++ {
		database, err = sql.Open("mysql", dsn)

		if err == nil {
			err = database.Ping()
		}

		if err == nil {
			log.Println("Connected to MySQL")
			return database
		}

		log.Printf(
			"MySQL connection failed (%d/%d): %v",
			i,
			maxRetries,
			err,
		)

		if database != nil {
			database.Close()
		}

		time.Sleep(retryDelay)
	}

	log.Fatalf(
		"Could not connect to MySQL after %d retries: %v",
		maxRetries,
		err,
	)

	return nil
}

func main() {
	// Environment variables
	mysqlUser := getEnv("MYSQL_USER", "root")
	mysqlPassword := getEnv("MYSQL_ROOT_PASSWORD", "password")
	mysqlHost := getEnv("MYSQL_HOST", "127.0.0.1")
	mysqlPort := getEnv("MYSQL_PORT", "3306")
	mysqlDatabase := getEnv("MYSQL_DATABASE", "db")

	// MySQL DSN
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true",
		mysqlUser,
		mysqlPassword,
		mysqlHost,
		mysqlPort,
		mysqlDatabase,
	)

	// Connect to MySQL with retry
	db = connectDB(dsn)
	defer db.Close()

	// Create users table if it does not exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255) NOT NULL
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Gin router
	r := gin.Default()

	r.GET("/users", getUsers)
	r.GET("/user/:user_id", getUser)
	r.POST("/user", createUser)

	log.Println("API server listening on :8080")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func getUsers(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, name, email
		FROM users
		ORDER BY id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	users := []User{}

	for rows.Next() {
		var user User

		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, users)
}

func getUser(c *gin.Context) {
	userID := c.Param("user_id")

	var user User

	err := db.QueryRow(`
		SELECT id, name, email
		FROM users
		WHERE id = ?
	`, userID).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

func createUser(c *gin.Context) {
	var user User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if user.Name == "" || user.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "name and email are required",
		})
		return
	}

	result, err := db.Exec(`
		INSERT INTO users (name, email)
		VALUES (?, ?)
	`, user.Name, user.Email)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	user.ID, err = result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}