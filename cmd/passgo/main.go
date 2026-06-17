package main

import (
	"crypto/rand"
	"log"
	"net/http"
	"os"

	"github.com/marekh19/passgo/internal/auth"
	apphttp "github.com/marekh19/passgo/internal/http"
	"github.com/marekh19/passgo/internal/session"
	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
	"github.com/marekh19/passgo/internal/transfer"
)

func main() {
	tbAddr := env("TB_ADDR", "127.0.0.1:33000")
	dbPath := env("DB_PATH", "dev.db")
	port := env("PORT", "8080")

	// tb -> store -> auth -> session -> transfer -> http
	tbc, err := tb.New(tbAddr)
	if err != nil {
		log.Fatalf("tb: %v", err)
	}
	defer tbc.Close()

	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	secret, secure := cookieConfig()
	authn := auth.New(secret, secure)
	sessions := session.New(tbc, st)
	transfers := transfer.New(tbc, st)
	srv := apphttp.New(st, sessions, transfers, authn)

	addr := ":" + port
	log.Printf("pass go listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func cookieConfig() (secret []byte, secure bool) {
	secure = os.Getenv("COOKIE_SECURE") == "true"
	if s := os.Getenv("COOKIE_SECRET"); s != "" {
		return []byte(s), secure
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("generate ephemeral cookie secret: %v", err)
	}
	log.Println("WARNING: COOKIE_SECRET unset; using a random ephemeral secret (cookies won't survive a restart)")
	return b, secure
}
