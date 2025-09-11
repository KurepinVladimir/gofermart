package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type resp struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
		order := r.URL.Path[len("/api/orders/"):]
		// простая логика:
		// если число чётное — PROCESSED с начислением; если оканчивается на 9 — INVALID;
		// иначе PROCESSING.
		if len(order) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		last := order[len(order)-1]
		switch last {
		case '9':
			json.NewEncoder(w).Encode(resp{Order: order, Status: "INVALID"})
		case '0', '2', '4', '6', '8':
			val := 123.45
			json.NewEncoder(w).Encode(resp{Order: order, Status: "PROCESSED", Accrual: &val})
		default:
			json.NewEncoder(w).Encode(resp{Order: order, Status: "PROCESSING"})
		}
	})
	s := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}
	log.Println("accrual stub on :8081")
	log.Fatal(s.ListenAndServe())
}
