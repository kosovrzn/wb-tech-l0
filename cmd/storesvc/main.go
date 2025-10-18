package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kosovrzn/wb-tech-l0/internal/cache"
	"github.com/kosovrzn/wb-tech-l0/internal/httpapi"
	"github.com/kosovrzn/wb-tech-l0/internal/kafkaconsumer"
	"github.com/kosovrzn/wb-tech-l0/internal/migrate"
	"github.com/kosovrzn/wb-tech-l0/internal/repo"

	"github.com/jackc/pgx/v5/pgxpool"
)

func redactDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.User != nil {
		if _, ok := u.User.Password(); ok {
			u.User = url.UserPassword(u.User.Username(), "*****")
		}
	}
	return u.String()
}

func main() {
	cfg := loadCfg()

	log.Printf("config: PG_DSN=%s KAFKA_BROKERS=%s KAFKA_TOPIC=%s KAFKA_GROUP=%s HTTP_ADDR=%s CACHE_CAPACITY=%d AUTO_MIGRATE=%t",
		redactDSN(cfg.PG_DSN), cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroup, cfg.HTTPAddr, cfg.CacheCapacity, cfg.AutoMigrate)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.PG_DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		log.Printf("running database migrations")
		if err := migrate.Up(ctx, cfg.PG_DSN); err != nil {
			log.Fatalf("migrations failed: %v", err)
		}
	}

	r := repo.NewPostgres(pool)

	c := cache.New(cfg.CacheCapacity)
	if cfg.WarmupLimit > 0 {
		rows, err := r.Warmup(ctx, cfg.WarmupLimit)
		if err != nil {
			log.Printf("warmup warn: %v", err)
		} else {
			for id, raw := range rows {
				c.Set(id, raw)
			}
			log.Printf("warmup: cached %d orders", len(rows))
		}
	}

	go func() {
		if err := kafkaconsumer.Run(ctx, cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroup, r, c); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("consumer stopped: %v", err)
			stop()
		}
	}()

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httpapi.NewHandler(r, c),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("HTTP listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http stopped: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Printf("bye")
}

type Cfg struct {
	PG_DSN        string
	KafkaBrokers  string
	KafkaTopic    string
	KafkaGroup    string
	HTTPAddr      string
	WarmupLimit   int
	CacheCapacity int
	AutoMigrate   bool
}

func loadCfg() Cfg {
	return Cfg{
		PG_DSN:        getenv("PG_DSN", "postgres://user:pass@localhost:5432/orders?sslmode=disable"),
		KafkaBrokers:  getenv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:    getenv("KAFKA_TOPIC", "orders"),
		KafkaGroup:    getenv("KAFKA_GROUP", "ordersvc"),
		HTTPAddr:      getenv("HTTP_ADDR", ":8081"),
		WarmupLimit:   getenvInt("WARMUP_LIMIT", 1000),
		CacheCapacity: getenvInt("CACHE_CAPACITY", 1000),
		AutoMigrate:   getenvBool("AUTO_MIGRATE", false),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
