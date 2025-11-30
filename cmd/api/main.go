package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/oggyb/insider-assessment/internal/cache/redis"
	"github.com/oggyb/insider-assessment/internal/config"
	"github.com/oggyb/insider-assessment/internal/db/gormdb"
	"github.com/oggyb/insider-assessment/internal/handler"
	mesgRepo "github.com/oggyb/insider-assessment/internal/repository/gorm/message"
	routes "github.com/oggyb/insider-assessment/internal/router"
	"github.com/oggyb/insider-assessment/internal/scheduler"
	"github.com/oggyb/insider-assessment/internal/server"
	"github.com/oggyb/insider-assessment/internal/service"
	"github.com/oggyb/insider-assessment/internal/sms"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Base context for the whole application lifetime.
	rootCtx := context.Background()

	// Load configuration from environment/.env.
	cfg := config.New()

	// Init cache.
	cache := redis.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := cache.Ping(rootCtx); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	// Init DB.
	dsn := cfg.PostgresDSN()
	db, err := gormdb.New(dsn)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}

	// Init SMS provider client.
	smsClient := sms.NewWebhookClient(cfg.SMS.ProviderURL, cfg.SMS.ProviderKey)
	if err := smsClient.Health(rootCtx); err != nil {
		log.Fatalf("failed to ping SMS provider: %v", err)
	}

	// Init repository and services.

	// Message
	msgRepository := mesgRepo.NewRepository(db)
	msgSvc := service.NewMessageService(
		msgRepository,
		smsClient,
		cache,
		cfg.Worker.BatchSize,
		cfg.Worker.MaxWorkers,
		cfg.Worker.PerMessageTimeout,
	)

	// Cron
	cron := scheduler.NewSchedulerService(
		msgSvc,
		cfg.Scheduler.Interval,
		cfg.Scheduler.BatchTimeout,
	)

	// HTTP dependencies & server wiring.

	// Handlers
	homeHandler := handler.NewHomeHandler()
	messageHandler := handler.NewMessageHandler(msgSvc, cron)

	// Init route dependencies
	deps := routes.AppDeps{
		Home:    homeHandler,
		Message: messageHandler,
	}

	// Init Server
	addr := fmt.Sprintf("%s:%s", cfg.API.Host, cfg.API.Port)
	srv := server.New(addr, deps)

	// Create a context that is cancelled on SIGINT/SIGTERM (Ctrl+C, docker stop etc.).
	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the HTTP server in a separate goroutine so we can listen for signals.
	go func() {
		log.Printf("HTTP server listening on %s", addr)

		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start the scheduler after everything is wired up.
	err = cron.Start()
	if err != nil {
		log.Fatalf("Cron job service error: %v", err)
	}
	log.Println("[Main] Scheduler started.")

	// Block until we receive a shutdown signal.
	<-ctx.Done()
	log.Println("[Main] Shutdown signal received, starting graceful shutdown...")

	// Give components some time to shut down cleanly.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop the scheduler (waits for in-flight batch to finish or timeout).
	log.Println("[Main] Stopping scheduler...")
	err = cron.Stop()
	if err != nil {
		log.Fatalf("Cron job could not stopped. error: %v", err)
	}
	log.Println("[Main] Scheduler stopped.")

	// Gracefully shut down the HTTP server.
	log.Println("[Main] Shutting down HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Main] HTTP server graceful shutdown failed: %v", err)
	} else {
		log.Println("[Main] HTTP server stopped.")
	}

	log.Println("[Main] Shutdown complete.")
}
