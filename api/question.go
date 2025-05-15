package main

type Question struct {
	ID       int    `json:"id"`
	Content  string `json:"content"`
	Answer   string `json:"answer"`
	Category string `json:"category"`
}
