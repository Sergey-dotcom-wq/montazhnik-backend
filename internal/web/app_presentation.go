package web

import (
	_ "embed"
)

// appPresentationHTML — презентация «Монтажник Связи 1.2», та же,
// что встроена в мобильное приложение (assets/presentation.html).
// Отдаётся на /presentation основного сервера (порт 8080), чтобы
// веб-дашборд открывал её без отдельного порта 8081.
//
//go:embed app_presentation.html
var appPresentationHTML string

// AppPresentationHTML — публичная переменная для маршрута.
var AppPresentationHTML = appPresentationHTML