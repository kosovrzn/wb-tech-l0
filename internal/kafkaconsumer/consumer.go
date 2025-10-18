package kafkaconsumer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kosovrzn/wb-tech-l0/internal/cache"
	"github.com/kosovrzn/wb-tech-l0/internal/domain"
	"github.com/kosovrzn/wb-tech-l0/internal/repo"
	"github.com/kosovrzn/wb-tech-l0/internal/validation"

	"github.com/segmentio/kafka-go"
)

// MessageReader abstracts kafka reader operations for easier testing.
//
//go:generate moq -pkg mocks -skip-ensure -out ../mocks/kafka_reader_mock.go . MessageReader
type MessageReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
	Close() error
}

func Run(ctx context.Context, brokers, topic, group string, r repo.Repository, c cache.Store) error {
	if err := ensureTopic(ctx, brokers, topic); err != nil {
		return err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               strings.Split(brokers, ","),
		GroupID:               group,
		GroupTopics:           []string{topic},
		StartOffset:           kafka.FirstOffset,
		MinBytes:              1,
		MaxBytes:              10 << 20,
		CommitInterval:        0,
		WatchPartitionChanges: true,
		Logger:                log.New(os.Stdout, "[kafka] ", 0),
		ErrorLogger:           log.New(os.Stderr, "[kafka-err] ", 0),
	})
	defer reader.Close()

	log.Printf("consumer START (brokers=%s topic=%s group=%s)", brokers, topic, group)

	return consume(ctx, reader, r, c, validation.New())
}

func consume(ctx context.Context, reader MessageReader, r repo.Repository, c cache.Store, validator *validation.Validator) error {
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
		if err := validator.ValidateOrder(&o); err != nil {
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

func ensureTopic(ctx context.Context, brokers, topic string) error {
	list := strings.Split(brokers, ",")
	if len(list) == 0 {
		return errors.New("no kafka brokers configured")
	}

	retryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var lastErr error
	for {
		select {
		case <-retryCtx.Done():
			if lastErr == nil {
				lastErr = retryCtx.Err()
			}
			return lastErr
		default:
		}

		conn, err := kafka.DialContext(retryCtx, "tcp", list[0])
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}

		err = conn.CreateTopics(kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
		conn.Close()
		if err != nil {
			if errors.Is(err, kafka.TopicAlreadyExists) {
				return nil
			}
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return nil
	}
}
