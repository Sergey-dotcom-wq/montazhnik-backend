package main

import (
	"log"
	"net/http"

	"github.com/yourorg/montazhnik/backend/internal/web"
)

// present — отдельный сайт презентации «Обзор платформы».
// Оригинальная презентация без изменений, на собственном порту 8081
// и под собственным названием — не пересекается с дашбордом (8080).
func main() {
	mux := http.NewServeMux()

	// Корень — полная инвестиционная презентация «портянкой».
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Write([]byte(web.InvestHTML))
	})

	log.Printf("Обзор платформы (презентация-портянка) слушает порт :8081")
	log.Fatal(http.ListenAndServe("0.0.0.0:8081", mux))
}