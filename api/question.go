package main

import (
	"time"
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
