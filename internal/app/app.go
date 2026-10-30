// Package app starts Habit: the world server and web client. It's shared by
// the root package (what `go build` builds, as hosts like Render run it) and
// cmd/habit.
package app

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // Plaza days run on New York time, even on servers without zone files

	"habit/internal/server"
	"habit/internal/store"
	"habit/internal/store/postgres"
	"habit/web"
)

// Main parses flags, opens the store and serves until SIGINT or SIGTERM.
func Main() {
	// Hosts like Render say which port to use in $PORT.
	defaultAddr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		defaultAddr = ":" + port
	}
	addr := flag.String("addr", defaultAddr, "address to listen on (default $PORT if set)")
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres URL, e.g. postgres://user:pass@localhost:5432/habit (default $DATABASE_URL; empty keeps everything in memory)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var st store.Store
	if *dbURL == "" {
		log.Print("No database set, so accounts and chat are kept in memory and lost on restart.")
		st = store.NewMemory()
	} else {
		pg, err := postgres.Open(ctx, *dbURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Print("Saving accounts and chat to Postgres.")
		st = pg
	}
	defer st.Close()

	sv, err := server.New(ctx, server.Options{Static: web.Static, Store: st})
	if err != nil {
		log.Fatal(err)
	}
	go sv.Run(ctx)

	srv := &http.Server{Addr: *addr, Handler: sv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()

	log.Printf("Habit is open at http://localhost%s", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
