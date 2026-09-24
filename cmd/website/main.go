package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/yourorg/montazhnik/backend/internal/web"
)

// website — отдельный публичный сайт-презентация «Монтажник Связи».
// «Фуршетный стол»: богатая витрина с презентацией, кнопкой подписки
// и реферальными ссылками. Работает на отдельном порту (8085),
// не пересекаясь с рабочим дашбордом (8080) и раздачей APK (8000).
func main() {
	port := os.Getenv("WEBSITE_PORT")
	if port == "" {
		port = "8085"
	}

	mux := http.NewServeMux()

	// Главная — лендинг с презентацией и реферальной привязкой.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ref := r.URL.Query().Get("ref")
		page := web.LandingHTML
		if ref != "" {
			// Если в URL есть реферальный код — подставляем его в кнопки
			// и сохраняем для клиента (через JS).
			page = strings.Replace(page, "{{REF}}", ref, -1)
		} else {
			page = strings.Replace(page, "{{REF}}", "", -1)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Write([]byte(page))
	})

	// Презентация (слайд-дек) — отдельная страница.
	mux.HandleFunc("/presentation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Write([]byte(web.PresentationHTML))
	})

	// Дашборд (рабочий вход коллектива) — редирект на основной сервер 8080.
	mux.HandleFunc("/cabinet", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://192.168.0.10:8080", http.StatusFound)
	})

	// Регистрация с реферальным кодом — переход на регистрацию в приложении.
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		ref := r.URL.Query().Get("ref")
		// Направляем на регистрацию; ref-код попадёт в форму (через localStorage на сайте).
		http.Redirect(w, r, "http://192.168.0.10:8080/#register?ref="+ref, http.StatusFound)
	})

	log.Printf("Website (лендинг) слушает порт :%s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, mux))
}