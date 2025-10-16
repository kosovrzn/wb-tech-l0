package kafkaconsumer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/kosovrzn/wb-tech-l0/internal/cache"
	"github.com/kosovrzn/wb-tech-l0/internal/domain"
	"github.com/kosovrzn/wb-tech-l0/internal/repo"
	"github.com/kosovrzn/wb-tech-l0/internal/validation"

	"github.com/segmentio/kafka-go"
)

func Run(ctx context.Context, brokers, topic, group string, r repo.Repository, c *cache.Cache) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(brokers, ","),
		Topic:          topic,
		GroupID:        group,
		MinBytes:       1,
		MaxBytes:       10 << 20,
		CommitInterval: 0,
		Logger:         log.New(os.Stdout, "[kafka] ", 0),
		ErrorLogger:    log.New(os.Stderr, "[kafka-err] ", 0),
	})

	log.Printf("consumer START (brokers=%s topic=%s group=%s)", brokers, topic, group)

	defer reader.Close()

	orderValidator := validation.New()

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		var o domain.Order
		if err := json.Unmarshal(m.Value, &o); err != nil {
			log.Printf("skip invalid msg: %v", err)
			_ = reader.CommitMessages(ctx, m)
			continue
		}
		if err := orderValidator.ValidateOrder(&o); err != nil {
			oid := o.OrderUID
			if oid == "" {
				oid = "<unknown>"
			}
			log.Printf("skip semantically invalid msg: order_uid=%s err=%v", oid, err)
			_ = reader.CommitMessages(ctx, m)
			continue
		}

		if err := r.UpsertOrder(ctx, &o, m.Value); err != nil {
			log.Printf("db upsert failed: %v", err)
			continue
		}
		c.Set(o.OrderUID, m.Value)

		if err := reader.CommitMessages(ctx, m); err != nil {
			log.Printf("commit failed: %v", err)
		}
	}
}
