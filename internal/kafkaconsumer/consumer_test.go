package kafkaconsumer

import (
	"context"
	"errors"
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/kosovrzn/wb-tech-l0/internal/domain"
	"github.com/kosovrzn/wb-tech-l0/internal/mocks"
	"github.com/kosovrzn/wb-tech-l0/internal/validation"
)

func TestConsume_ProcessesValidMessage(t *testing.T) {
	ctx := context.Background()

	validMsg := kafka.Message{Value: []byte(`{
		"order_uid": "ORDER1",
		"track_number": "TRACK123456",
		"entry": "WBIL",
		"delivery": {
			"name": "John Doe",
			"phone": "+10000000000",
			"zip": "123456",
			"city": "City",
			"address": "Street 1",
			"region": "Region",
			"email": "john@example.com"
		},
		"payment": {
			"transaction": "TXN1",
			"request_id": "req",
			"currency": "USD",
			"provider": "wbpay",
			"amount": 100,
			"payment_dt": 1637907727,
			"bank": "alpha",
			"delivery_cost": 10,
			"goods_total": 90,
			"custom_fee": 0
		},
		"items": [{
			"chrt_id": 1,
			"track_number": "TRACK123456",
			"price": 90,
			"rid": "RID1",
			"name": "Item",
			"sale": 0,
			"size": "M",
			"total_price": 90,
			"nm_id": 1001,
			"brand": "Brand",
			"status": 200
		}],
		"locale": "EN",
		"internal_signature": "",
		"customer_id": "cust1",
		"delivery_service": "meest",
		"shardkey": "1",
		"sm_id": 99,
		"date_created": "2021-11-26T06:22:19Z",
		"oof_shard": "1"
	}`)}
	messages := []kafka.Message{validMsg}

	readerMock := &mocks.MessageReaderMock{}
	readerMock.FetchMessageFunc = func(context.Context) (kafka.Message, error) {
		if len(messages) == 0 {
			return kafka.Message{}, context.Canceled
		}
		m := messages[0]
		messages = messages[1:]
		return m, nil
	}
	commitCalled := false
	readerMock.CommitMessagesFunc = func(_ context.Context, msgs ...kafka.Message) error {
		commitCalled = true
		if len(msgs) != 1 {
			t.Fatalf("expected commit of 1 message, got %d", len(msgs))
		}
		return nil
	}
	readerMock.CloseFunc = func() error { return nil }

	upsertCalled := false
	repoMock := &mocks.RepositoryMock{}
	repoMock.UpsertOrderFunc = func(context.Context, *domain.Order, []byte) error {
		upsertCalled = true
		return nil
	}

	cacheStored := false
	cacheMock := &mocks.StoreMock{
		GetFunc: func(string) ([]byte, bool) { return nil, false },
		SetFunc: func(id string, _ []byte) { cacheStored = (id == "ORDER1") },
	}

	err := consume(ctx, readerMock, repoMock, cacheMock, validation.New())
	if err != nil {
		t.Fatalf("consume returned error: %v", err)
	}
	if !upsertCalled {
		t.Fatalf("expected upsert to be called")
	}
	if !cacheStored {
		t.Fatalf("expected cache to store value")
	}
	if !commitCalled {
		t.Fatalf("expected commit to be called")
	}
}

func TestConsume_DoesNotCommitOnUpsertError(t *testing.T) {
	ctx := context.Background()

	validMsg := kafka.Message{Value: []byte(`{
		"order_uid": "ORDER2",
		"track_number": "TRACKX",
		"entry": "WBIL",
		"delivery": {
			"name": "Jane",
			"phone": "+10000000001",
			"zip": "654321",
			"city": "City",
			"address": "Street 2",
			"region": "Region",
			"email": "jane@example.com"
		},
		"payment": {
			"transaction": "TXN2",
			"currency": "USD",
			"provider": "wbpay",
			"amount": 200,
			"payment_dt": 1637907727,
			"bank": "alpha",
			"delivery_cost": 20,
			"goods_total": 180,
			"custom_fee": 0
		},
		"items": [{
			"chrt_id": 2,
			"track_number": "TRACKX",
			"price": 180,
			"rid": "RID2",
			"name": "Item2",
			"sale": 0,
			"size": "L",
			"total_price": 180,
			"nm_id": 1002,
			"brand": "Brand",
			"status": 200
		}],
		"locale": "EN",
		"customer_id": "cust2",
		"delivery_service": "meest",
		"shardkey": "1",
		"sm_id": 100,
		"date_created": "2021-11-26T06:22:19Z",
		"oof_shard": "1"
	}`)}

	messages := []kafka.Message{validMsg}
	readerMock := &mocks.MessageReaderMock{}
	readerMock.FetchMessageFunc = func(context.Context) (kafka.Message, error) {
		if len(messages) == 0 {
			return kafka.Message{}, context.Canceled
		}
		m := messages[0]
		messages = messages[1:]
		return m, nil
	}
	readerMock.CommitMessagesFunc = func(context.Context, ...kafka.Message) error {
		t.Fatalf("commit should not be called when upsert fails")
		return nil
	}
	readerMock.CloseFunc = func() error { return nil }

	repoMock := &mocks.RepositoryMock{}
	repoMock.UpsertOrderFunc = func(context.Context, *domain.Order, []byte) error {
		return errors.New("db error")
	}

	cacheMock := &mocks.StoreMock{
		GetFunc: func(string) ([]byte, bool) { return nil, false },
		SetFunc: func(string, []byte) { t.Fatalf("cache should not be set when upsert fails") },
	}

	err := consume(ctx, readerMock, repoMock, cacheMock, validation.New())
	if err != nil {
		t.Fatalf("consume returned error: %v", err)
	}
}

func TestConsume_CommitsInvalidMessage(t *testing.T) {
	ctx := context.Background()

	messages := []kafka.Message{{Value: []byte("invalid json")}}

	readerMock := &mocks.MessageReaderMock{}
	readerMock.FetchMessageFunc = func(context.Context) (kafka.Message, error) {
		if len(messages) == 0 {
			return kafka.Message{}, context.Canceled
		}
		m := messages[0]
		messages = messages[1:]
		return m, nil
	}
	commitCount := 0
	readerMock.CommitMessagesFunc = func(context.Context, ...kafka.Message) error {
		commitCount++
		return nil
	}
	readerMock.CloseFunc = func() error { return nil }

	repoMock := &mocks.RepositoryMock{}
	repoMock.UpsertOrderFunc = func(context.Context, *domain.Order, []byte) error {
		t.Fatalf("upsert should not be called on invalid JSON")
		return nil
	}

	cacheMock := &mocks.StoreMock{
		GetFunc: func(string) ([]byte, bool) { return nil, false },
		SetFunc: func(string, []byte) { t.Fatalf("cache should not be set on invalid message") },
	}

	err := consume(ctx, readerMock, repoMock, cacheMock, validation.New())
	if err != nil {
		t.Fatalf("consume returned error: %v", err)
	}
	if commitCount != 1 {
		t.Fatalf("expected commit to be called once, got %d", commitCount)
	}
}
