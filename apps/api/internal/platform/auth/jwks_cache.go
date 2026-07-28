package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"golang.org/x/sync/singleflight"
)

type jwksCache struct {
	url           string
	client        *http.Client
	maxBodyLength int64
	maxKeys       int
	ttl           time.Duration
	now           func() time.Time

	mu            sync.RWMutex
	keySet        jwk.Set
	expiresAt     time.Time
	lastRefreshAt time.Time

	group singleflight.Group
}

const minRefreshInterval = 5 * time.Second

func newJWKSCache(url string, client *http.Client, maxBodyLength int64, maxKeys int, ttl time.Duration, now func() time.Time) *jwksCache {
	return &jwksCache{
		url:           url,
		client:        client,
		maxBodyLength: maxBodyLength,
		maxKeys:       maxKeys,
		ttl:           ttl,
		now:           now,
	}
}

func (c *jwksCache) GetKey(ctx context.Context, kid string) (jwk.Key, error) {
	c.mu.RLock()
	set := c.keySet
	expires := c.expiresAt
	lastRefresh := c.lastRefreshAt
	c.mu.RUnlock()

	now := c.now()
	valid := set != nil && now.Before(expires)

	if valid {
		if key, found := set.LookupKeyID(kid); found {
			return c.validateKey(key)
		}
	}

	// If valid, and we refreshed very recently, do not refresh again to avoid storms from unknown kids.
	if valid && now.Sub(lastRefresh) < minRefreshInterval {
		return nil, ErrInvalidToken
	}

	// We either have an expired cache or a missing kid that hasn't triggered a refresh recently.
	val, err, _ := c.group.Do("refresh", func() (interface{}, error) {
		return c.refresh(ctx)
	})

	if err != nil {
		if valid {
			if key, found := set.LookupKeyID(kid); found {
				return c.validateKey(key)
			}
			return nil, ErrInvalidToken
		}
		return nil, ErrTokenVerificationUnavailable
	}

	newSet := val.(jwk.Set)
	if key, found := newSet.LookupKeyID(kid); found {
		return c.validateKey(key)
	}

	return nil, ErrInvalidToken
}

func (c *jwksCache) validateKey(key jwk.Key) (jwk.Key, error) {
	if algorithm, hasAlgorithm := key.Algorithm(); hasAlgorithm && algorithm != jwa.ES256() {
		return nil, ErrInvalidToken
	}
	return key, nil
}

func (c *jwksCache) refresh(ctx context.Context) (jwk.Set, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/jwk-set+json, application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status code")
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, c.maxBodyLength+1))
	if err != nil || int64(len(body)) > c.maxBodyLength || len(body) == 0 {
		return nil, errors.New("invalid body")
	}

	set, err := jwk.Parse(body, jwk.WithMaxKeys(c.maxKeys), jwk.WithRejectDuplicateKID(true))
	if err != nil || set.Len() == 0 {
		return nil, errors.New("invalid jwks")
	}

	validSet := jwk.NewSet()
	for i := 0; i < set.Len(); i++ {
		key, _ := set.Key(i)
		alg, hasAlg := key.Algorithm()
		if hasAlg && alg != jwa.ES256() {
			continue // Skip invalid algs
		}

		if keyType := key.KeyType(); keyType != jwa.EC() {
			continue
		}
		validSet.AddKey(key)
	}

	if validSet.Len() == 0 {
		return nil, errors.New("no valid keys found")
	}

	c.mu.Lock()
	c.keySet = validSet
	c.expiresAt = c.now().Add(c.ttl)
	c.lastRefreshAt = c.now()
	c.mu.Unlock()

	return validSet, nil
}
