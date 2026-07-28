package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
)

func TestJWKSCache(t *testing.T) {
	key1 := newSigningKey(t, "key-1")
	key2 := newSigningKey(t, "key-2")

	var requests atomic.Int32
	var currentKey atomic.Value
	currentKey.Store(key1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writeJWKS(t, w, currentKey.Load().(jwk.Key))
	}))
	defer server.Close()

	var currentTime time.Time = time.Now()
	nowFn := func() time.Time {
		return currentTime
	}

	cache := newJWKSCache(server.URL, server.Client(), 64*1024, 16, time.Minute, nowFn)

	t.Run("first request fetches from network", func(t *testing.T) {
		requests.Store(0)
		_, err := cache.GetKey(context.Background(), "key-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requests.Load() != 1 {
			t.Fatalf("expected 1 request, got %d", requests.Load())
		}
	})

	t.Run("cache hit avoids network request", func(t *testing.T) {
		requests.Store(0)
		_, err := cache.GetKey(context.Background(), "key-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requests.Load() != 0 {
			t.Fatalf("expected 0 requests for cache hit, got %d", requests.Load())
		}
	})

	t.Run("unknown kid triggers refresh", func(t *testing.T) {
		requests.Store(0)
		// We advance time just enough to bypass the minRefreshInterval to allow refresh.
		currentTime = currentTime.Add(10 * time.Second)
		currentKey.Store(key2) // server now serves key-2

		_, err := cache.GetKey(context.Background(), "key-2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requests.Load() != 1 {
			t.Fatalf("expected 1 request for unknown kid, got %d", requests.Load())
		}
	})

	t.Run("unknown kid repeatedly within min refresh interval does not spam", func(t *testing.T) {
		requests.Store(0)
		// we just refreshed (time is still exactly as left by previous test)
		_, err := cache.GetKey(context.Background(), "missing-key")
		if err != ErrInvalidToken {
			t.Fatalf("expected ErrInvalidToken, got %v", err)
		}
		if requests.Load() != 0 {
			t.Fatalf("expected 0 requests due to rate limiting, got %d", requests.Load())
		}
	})

	t.Run("cache expiration fail closed", func(t *testing.T) {
		requests.Store(0)
		// Expire the cache
		currentTime = currentTime.Add(2 * time.Minute)
		// Make server return 500
		currentKey.Store(key1)

		brokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer brokenServer.Close()

		brokenCache := newJWKSCache(brokenServer.URL, brokenServer.Client(), 64*1024, 16, time.Minute, nowFn)
		// prime cache (simulate it was valid)
		brokenCache.keySet = jwk.NewSet()
		key1Public, _ := key1.PublicKey()
		_ = brokenCache.keySet.AddKey(key1Public)
		brokenCache.expiresAt = currentTime.Add(-time.Minute) // expired!

		_, err := brokenCache.GetKey(context.Background(), "key-1")
		if err != ErrTokenVerificationUnavailable {
			t.Fatalf("expected ErrTokenVerificationUnavailable for expired cache and broken network, got %v", err)
		}
	})
}

func TestJWKSCacheConcurrency(t *testing.T) {
	key := newSigningKey(t, "key-1")
	var requests atomic.Int32
	var inFlight atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		in := inFlight.Add(1)
		if in > 1 {
			t.Errorf("expected max 1 in-flight request, got %d", in)
		}
		requests.Add(1)
		time.Sleep(50 * time.Millisecond) // artificially delay to guarantee concurrency collision
		inFlight.Add(-1)
		writeJWKS(t, w, key)
	}))
	defer server.Close()

	cache := newJWKSCache(server.URL, server.Client(), 64*1024, 16, time.Minute, time.Now)

	var wg sync.WaitGroup
	const numGoroutines = 50
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := cache.GetKey(context.Background(), "key-1")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	if requests.Load() != 1 {
		t.Fatalf("expected exactly 1 request due to singleflight, got %d", requests.Load())
	}
}
