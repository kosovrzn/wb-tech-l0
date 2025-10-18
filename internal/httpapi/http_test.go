package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kosovrzn/wb-tech-l0/internal/httpapi"
	"github.com/kosovrzn/wb-tech-l0/internal/mocks"
)

func TestOrderHandlerReturnsCachedValue(t *testing.T) {
	raw := []byte(`{"order_uid":"cached"}`)

	cacheMock := &mocks.StoreMock{
		GetFunc: func(id string) ([]byte, bool) {
			if id != "cached" {
				t.Fatalf("unexpected cache key: %s", id)
			}
			return raw, true
		},
		SetFunc: func(string, []byte) {
			t.Fatalf("unexpected cache Set call")
		},
	}
	repoMock := &mocks.RepositoryMock{}
	repoMock.GetOrderRawFunc = func(ctx context.Context, id string) ([]byte, error) {
		t.Fatalf("repository should not be called on cache hit")
		return nil, nil
	}

	handler := httpapi.NewHandler(repoMock, cacheMock)
	req := httptest.NewRequest(http.MethodGet, "/order/cached", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	if got := res.Header.Get("X-Cache"); got != "HIT" {
		t.Fatalf("expected X-Cache=HIT, got %s", got)
	}

	body := rec.Body.Bytes()
	if !bytes.Equal(body, raw) {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestOrderHandlerLoadsFromRepoOnCacheMiss(t *testing.T) {
	raw := []byte(`{"order_uid":"repo"}`)
	cacheStored := false

	cacheMock := &mocks.StoreMock{
		GetFunc: func(id string) ([]byte, bool) {
			return nil, false
		},
		SetFunc: func(id string, b []byte) {
			cacheStored = true
			if id != "repo" {
				t.Fatalf("unexpected id in cache set: %s", id)
			}
			if !bytes.Equal(b, raw) {
				t.Fatalf("cache stored unexpected data: %s", string(b))
			}
		},
	}

	repoMock := &mocks.RepositoryMock{}
	repoMock.GetOrderRawFunc = func(ctx context.Context, id string) ([]byte, error) {
		if id != "repo" {
			t.Fatalf("unexpected repo id: %s", id)
		}
		if ctx == nil {
			t.Fatalf("expected context to be passed")
		}
		return raw, nil
	}

	handler := httpapi.NewHandler(repoMock, cacheMock)
	req := httptest.NewRequest(http.MethodGet, "/order/repo", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	if got := res.Header.Get("X-Cache"); got != "MISS" {
		t.Fatalf("expected X-Cache=MISS, got %s", got)
	}
	if !cacheStored {
		t.Fatalf("expected value to be stored in cache")
	}
	if !bytes.Equal(rec.Body.Bytes(), raw) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestOrderHandlerHandlesMissingOrder(t *testing.T) {
	cacheMock := &mocks.StoreMock{
		GetFunc: func(string) ([]byte, bool) { return nil, false },
		SetFunc: func(string, []byte) {},
	}
	repoMock := &mocks.RepositoryMock{}
	repoMock.GetOrderRawFunc = func(ctx context.Context, id string) ([]byte, error) {
		return nil, errors.New("not found")
	}

	handler := httpapi.NewHandler(repoMock, cacheMock)
	req := httptest.NewRequest(http.MethodGet, "/order/missing", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
