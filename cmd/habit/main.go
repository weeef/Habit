// Command habit runs the Habit prototype: the world server and web client.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"habit/internal/server"
	"habit/web"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sv := server.New(server.Options{Static: web.Static})
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
