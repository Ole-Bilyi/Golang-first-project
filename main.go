package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

type Question struct {
	ID        int       `json:"id"`
	TextA     string    `json:"text_a"` // Pytanie główne (3 pkt)
	HintB     string    `json:"hint_b"` // Podpowiedź 1 (2 pkt)
	HintC     string    `json:"hint_c"` // Podpowiedź 2 (1 pkt)
	Answer    string    `json:"answer"` // Poprawna odpowiedź
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type QueryRequest struct {
	SearchText string `json:"search_text"`
	Field      string `json:"field"`     // which field to search in (text_a, hint_b, hint_c, answer)
	OrderBy    string `json:"order_by"`  // field to order by
	OrderDir   string `json:"order_dir"` // asc or desc
	Limit      int    `json:"limit"`     // max number of results
}

var db *sql.DB
var tmpl *template.Template

// Middleware for logging HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// Middleware for panic recovery
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Middleware for basic security headers
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// validateQuestion checks if a question meets all requirements
func validateQuestion(q *Question) error {
	if len(q.TextA) < 3 {
		return fmt.Errorf("text_a must be at least 3 characters long")
	}
	if len(q.Answer) < 1 {
		return fmt.Errorf("answer must not be empty")
	}
	return nil
}

func main() {
	var err error
	// Properly assign to global db variable
	db, err = sql.Open("sqlite", "./db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test database connection
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Create table if not exists
	createTable := `
	CREATE TABLE IF NOT EXISTS questions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		text_a TEXT NOT NULL,
		hint_b TEXT,
		hint_c TEXT,
		answer TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	// Parse templates
	tmpl, err = template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal("Failed to parse template:", err)
	}

	r := mux.NewRouter()

	// Apply middleware
	r.Use(loggingMiddleware)
	r.Use(recoveryMiddleware)
	r.Use(securityHeadersMiddleware)

	// Serve admin page
	r.HandleFunc("/", adminPageHandler).Methods("GET")

	// Serve static CSS with cache headers
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	r.PathPrefix("/static/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		staticHandler.ServeHTTP(w, r)
	}))

	// API endpoints
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/questions", getQuestionsHandler).Methods("GET")
	api.HandleFunc("/questions", createQuestionHandler).Methods("POST")
	api.HandleFunc("/questions/{id}", updateQuestionHandler).Methods("PUT")
	api.HandleFunc("/questions/{id}", deleteQuestionHandler).Methods("DELETE")
	api.HandleFunc("/search", searchQuestionsHandler).Methods("POST")

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Server started on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited gracefully")
}

func adminPageHandler(w http.ResponseWriter, r *http.Request) {
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
	}
}

func getQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, text_a, hint_b, hint_c, answer, created_at, updated_at 
		FROM questions 
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		log.Printf("Database query error: %v", err)
		return
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		err := rows.Scan(&q.ID, &q.TextA, &q.HintB, &q.HintC, &q.Answer, &q.CreatedAt, &q.UpdatedAt)
		if err != nil {
			http.Error(w, "Failed to scan database row", http.StatusInternalServerError)
			log.Printf("Row scan error: %v", err)
			return
		}
		questions = append(questions, q)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Error iterating over rows", http.StatusInternalServerError)
		log.Printf("Row iteration error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(questions); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func createQuestionHandler(w http.ResponseWriter, r *http.Request) {
	var q Question
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("JSON decode error: %v", err)
		return
	}

	if err := validateQuestion(&q); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO questions (text_a, hint_b, hint_c, answer, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, q.TextA, q.HintB, q.HintC, q.Answer, now, now)
	if err != nil {
		http.Error(w, "Failed to create question", http.StatusInternalServerError)
		log.Printf("Database insert error: %v", err)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to get inserted ID", http.StatusInternalServerError)
		log.Printf("LastInsertId error: %v", err)
		return
	}

	q.ID = int(id)
	q.CreatedAt = now
	q.UpdatedAt = now

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(q); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func updateQuestionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var q Question
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("JSON decode error: %v", err)
		return
	}

	if err := validateQuestion(&q); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now()
	result, err := db.Exec(`
		UPDATE questions 
		SET text_a=?, hint_b=?, hint_c=?, answer=?, updated_at=? 
		WHERE id=?
	`, q.TextA, q.HintB, q.HintC, q.Answer, now, id)
	if err != nil {
		http.Error(w, "Failed to update question", http.StatusInternalServerError)
		log.Printf("Database update error: %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to get rows affected", http.StatusInternalServerError)
		log.Printf("RowsAffected error: %v", err)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Question not found", http.StatusNotFound)
		return
	}

	// Fetch the updated question to get the correct timestamps
	err = db.QueryRow(`
		SELECT id, text_a, hint_b, hint_c, answer, created_at, updated_at 
		FROM questions 
		WHERE id = ?
	`, id).Scan(&q.ID, &q.TextA, &q.HintB, &q.HintC, &q.Answer, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		http.Error(w, "Failed to fetch updated question", http.StatusInternalServerError)
		log.Printf("Database query error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(q); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func deleteQuestionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	result, err := db.Exec("DELETE FROM questions WHERE id=?", id)
	if err != nil {
		http.Error(w, "Failed to delete question", http.StatusInternalServerError)
		log.Printf("Database delete error: %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to get rows affected", http.StatusInternalServerError)
		log.Printf("RowsAffected error: %v", err)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Question not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func searchQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	var query QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("JSON decode error: %v", err)
		return
	}

	// Validate fields
	allowedFields := map[string]bool{
		"text_a": true,
		"hint_b": true,
		"hint_c": true,
		"answer": true,
	}

	if !allowedFields[query.Field] {
		http.Error(w, "Invalid field specified", http.StatusBadRequest)
		return
	}

	// Validate order direction
	query.OrderDir = strings.ToUpper(query.OrderDir)
	if query.OrderDir != "ASC" && query.OrderDir != "DESC" {
		query.OrderDir = "ASC"
	}

	// Validate and set limit
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 10 // default limit
	}

	// Build the query safely using parameterized queries
	sqlQuery := fmt.Sprintf(`
		SELECT id, text_a, hint_b, hint_c, answer, created_at, updated_at 
		FROM questions 
		WHERE %s LIKE ? 
		ORDER BY %s %s
		LIMIT ?
	`, query.Field, query.Field, query.OrderDir)

	// Execute the query
	rows, err := db.Query(sqlQuery, "%"+query.SearchText+"%", query.Limit)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		log.Printf("Database query error: %v", err)
		return
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		err := rows.Scan(&q.ID, &q.TextA, &q.HintB, &q.HintC, &q.Answer, &q.CreatedAt, &q.UpdatedAt)
		if err != nil {
			http.Error(w, "Failed to scan database row", http.StatusInternalServerError)
			log.Printf("Row scan error: %v", err)
			return
		}
		questions = append(questions, q)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Error iterating over rows", http.StatusInternalServerError)
		log.Printf("Row iteration error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(questions); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}
