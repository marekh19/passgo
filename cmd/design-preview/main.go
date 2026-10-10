package main

import (
	"log"
	"net/http"

	"github.com/marekh19/passgo/internal/views"
)

func previewHandler(static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", static))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.DesignPreview().Render(r.Context(), w); err != nil {
			log.Printf("render preview: %v", err)
		}
	})
	return mux
}

func main() {
	log.Print("Design preview: http://<server-lan-ip>:7119 (all interfaces; trusted networks only)")
	log.Fatal(http.ListenAndServe("0.0.0.0:7119", previewHandler(http.FileServer(http.Dir("static")))))
}
