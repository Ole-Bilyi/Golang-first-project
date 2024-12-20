package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

const (
	host     = "cns01.h.filess.io"
	port     = "3305"
	user     = "GoDB_problemran"
	password = "cf5b5bbff374a19e8dc1842b57857024df6ebb75"
	dbname   = "GoDB_problemran"
)

type Record struct {
	ID    int
	Names string
}

func main() {
	// Serve static files (CSS/JS/images)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Route for home page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Fetch data from database
		data, err := fetchDataFromDB()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Parse and execute template
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, data)
	})

	log.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Fetch data from the database
func fetchDataFromDB() ([]Record, error) {
	uri := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, dbname)

	db, err := sql.Open("mysql", uri)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, names FROM main_table")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var record Record
		if err := rows.Scan(&record.ID, &record.Names); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}
