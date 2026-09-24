package web

import (
	_ "embed"
)

// originalPresentationHTML — ОРИГИНАЛ презентации, без изменений.
// Файл скопирован 1-в-1 из исходного проекта (SHA256 70B0C429F53B0C88),
// встроен в бинарник через go:embed. Содержимое не трогаем.
//
//go:embed original_presentation.html
var originalPresentationHTML string

// PresentationHTML отдаёт оригинальную презентацию как есть.
var PresentationHTML = originalPresentationHTML