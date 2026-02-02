package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	db  *sql.DB
	rdb *redis.Client
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/users_db?sslmode=disable"
	}

	var err error
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Initialize schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Connect to Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	opt, _ := redis.ParseURL(redisURL)
	rdb = redis.NewClient(opt)
	log.Println("Connected to Redis")

	// Routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/users", usersHandler)
	http.HandleFunc("/users/", userHandler)

	log.Printf("user-service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "user-service"})
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		rows, err := db.Query("SELECT id, email, name, created_at FROM users ORDER BY id")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var u User
			rows.Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
			users = append(users, u)
		}
		json.NewEncoder(w).Encode(users)

	case "POST":
		var u User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := db.QueryRow(
			"INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id, created_at",
			u.Email, u.Name,
		).Scan(&u.ID, &u.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Cache in Redis
		rdb.Set(r.Context(), fmt.Sprintf("user:%d", u.ID), u.Email, time.Hour)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(u)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path /users/{id}
	id := r.URL.Path[len("/users/"):]

	var u User
	err := db.QueryRow("SELECT id, email, name, created_at FROM users WHERE id = $1", id).
		Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(u)
}
