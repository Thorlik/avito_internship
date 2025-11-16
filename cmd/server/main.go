package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Thorlik/avito_internship/internal/app/config"
	"github.com/Thorlik/avito_internship/internal/app/handlers"
	"github.com/Thorlik/avito_internship/internal/domain/service"
	"github.com/Thorlik/avito_internship/internal/infrastructure/persistence"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
	}

	var store *persistence.PostgresStorage
	for i := 0; i < 10; i++ {
		store, err = persistence.NewPostgresStorage(cfg.GetDSN())
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database (attempt %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("Failed to connect to database after retries: %v", err)
	}
	defer store.Close()

	svc := service.NewService(store)
	handler := handlers.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/team/add", handler.CreateTeam)
	mux.HandleFunc("/team/get", handler.GetTeam)
	mux.HandleFunc("/users/setIsActive", handler.SetUserActive)
	mux.HandleFunc("/users/getReview", handler.GetUserReviews)
	mux.HandleFunc("/pullRequest/create", handler.CreatePullRequest)
	mux.HandleFunc("/pullRequest/merge", handler.MergePullRequest)
	mux.HandleFunc("/pullRequest/reassign", handler.ReassignReviewer)
	mux.HandleFunc("/statistics", handler.GetStatistics)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting server on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}
