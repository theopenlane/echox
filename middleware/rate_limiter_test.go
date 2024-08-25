package middleware

import (
	"errors"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/theopenlane/echox"
)

func TestRateLimiter(t *testing.T) {
	e := echox.New()

	handler := func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	}

	var inMemoryStore = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 3})

	mw := RateLimiterWithConfig(RateLimiterConfig{Store: inMemoryStore})

	testCases := []struct {
		id        string
		expectErr string
	}{
		{id: "127.0.0.1"},
		{id: "127.0.0.1"},
		{id: "127.0.0.1"},
		{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
		{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
		{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
		{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Add(echox.HeaderXRealIP, tc.id)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := mw(handler)(c)
		if tc.expectErr != "" {
			assert.EqualError(t, err, tc.expectErr)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestMustRateLimiterWithConfig_panicBehaviour(t *testing.T) {
	var inMemoryStore = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 3})

	assert.Panics(t, func() {
		RateLimiterWithConfig(RateLimiterConfig{})
	})

	assert.NotPanics(t, func() {
		RateLimiterWithConfig(RateLimiterConfig{Store: inMemoryStore})
	})
}

func TestRateLimiterWithConfig(t *testing.T) {
	var inMemoryStore = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 3})

	e := echox.New()

	handler := func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	}

	mw, err := RateLimiterConfig{
		IdentifierExtractor: func(c echox.Context) (string, error) {
			id := c.Request().Header.Get(echox.HeaderXRealIP)
			if id == "" {
				return "", errors.New("invalid identifier")
			}
			return id, nil
		},
		DenyHandler: func(ctx echox.Context, identifier string, err error) error {
			return ctx.JSON(http.StatusForbidden, nil)
		},
		ErrorHandler: func(ctx echox.Context, err error) error {
			return ctx.JSON(http.StatusBadRequest, nil)
		},
		Store: inMemoryStore,
	}.ToMiddleware()
	assert.NoError(t, err)

	testCases := []struct {
		id   string
		code int
	}{
		{"127.0.0.1", http.StatusOK},
		{"127.0.0.1", http.StatusOK},
		{"127.0.0.1", http.StatusOK},
		{"127.0.0.1", http.StatusForbidden},
		{"", http.StatusBadRequest},
		{"127.0.0.1", http.StatusForbidden},
		{"127.0.0.1", http.StatusForbidden},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Add(echox.HeaderXRealIP, tc.id)

		rec := httptest.NewRecorder()

		c := e.NewContext(req, rec)

		err := mw(handler)(c)

		assert.NoError(t, err)
		assert.Equal(t, tc.code, rec.Code)
	}
}

func TestRateLimiterWithConfig_defaultDenyHandler(t *testing.T) {
	var inMemoryStore = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 3})

	e := echox.New()

	handler := func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	}

	mw, err := RateLimiterConfig{
		IdentifierExtractor: func(c echox.Context) (string, error) {
			id := c.Request().Header.Get(echox.HeaderXRealIP)
			if id == "" {
				return "", errors.New("invalid identifier")
			}
			return id, nil
		},
		Store: inMemoryStore,
	}.ToMiddleware()
	assert.NoError(t, err)

	testCases := []struct {
		id        string
		expectErr string
	}{
		{id: "127.0.0.1"},
		{id: "127.0.0.1"},
		{id: "127.0.0.1"},
		{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
		{expectErr: "code=403, message=error while extracting identifier, internal=invalid identifier"},
		{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
		{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Add(echox.HeaderXRealIP, tc.id)

		rec := httptest.NewRecorder()

		c := e.NewContext(req, rec)

		err := mw(handler)(c)
		if tc.expectErr != "" {
			assert.EqualError(t, err, tc.expectErr)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimiterWithConfig_defaultConfig(t *testing.T) {
	{
		var inMemoryStore = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 3})

		e := echox.New()

		handler := func(c echox.Context) error {
			return c.String(http.StatusOK, "test")
		}

		mw, err := RateLimiterConfig{
			Store: inMemoryStore,
		}.ToMiddleware()
		assert.NoError(t, err)

		testCases := []struct {
			id        string
			expectErr string
		}{
			{id: "127.0.0.1"},
			{id: "127.0.0.1"},
			{id: "127.0.0.1"},
			{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
			{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
			{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
			{id: "127.0.0.1", expectErr: "code=429, message=rate limit exceeded"},
		}

		for _, tc := range testCases {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Add(echox.HeaderXRealIP, tc.id)

			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)

			err := mw(handler)(c)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, http.StatusOK, rec.Code)
		}
	}
}

func TestRateLimiterWithConfig_skipper(t *testing.T) {
	e := echox.New()

	var beforeFuncRan bool
	handler := func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	}
	var inMemoryStore = NewRateLimiterMemoryStore(5)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(echox.HeaderXRealIP, "127.0.0.1")

	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)

	mw, err := RateLimiterConfig{
		Skipper: func(c echox.Context) bool {
			return true
		},
		BeforeFunc: func(c echox.Context) {
			beforeFuncRan = true
		},
		Store: inMemoryStore,
		IdentifierExtractor: func(ctx echox.Context) (string, error) {
			return "127.0.0.1", nil
		},
	}.ToMiddleware()
	assert.NoError(t, err)

	err = mw(handler)(c)

	assert.NoError(t, err)
	assert.Equal(t, false, beforeFuncRan)
}

func TestRateLimiterWithConfig_skipperNoSkip(t *testing.T) {
	e := echox.New()

	var beforeFuncRan bool
	handler := func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	}
	var inMemoryStore = NewRateLimiterMemoryStore(5)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(echox.HeaderXRealIP, "127.0.0.1")

	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)

	mw, err := RateLimiterConfig{
		Skipper: func(c echox.Context) bool {
			return false
		},
		BeforeFunc: func(c echox.Context) {
			beforeFuncRan = true
		},
		Store: inMemoryStore,
		IdentifierExtractor: func(ctx echox.Context) (string, error) {
			return "127.0.0.1", nil
		},
	}.ToMiddleware()
	assert.NoError(t, err)

	_ = mw(handler)(c)

	assert.Equal(t, true, beforeFuncRan)
}

func TestRateLimiterWithConfig_beforeFunc(t *testing.T) {
	e := echox.New()

	handler := func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	}

	var beforeRan bool
	var inMemoryStore = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 3})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(echox.HeaderXRealIP, "127.0.0.1")

	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)

	mw, err := RateLimiterConfig{
		BeforeFunc: func(c echox.Context) {
			beforeRan = true
		},
		Store: inMemoryStore,
		IdentifierExtractor: func(ctx echox.Context) (string, error) {
			return "127.0.0.1", nil
		},
	}.ToMiddleware()
	assert.NoError(t, err)

	err = mw(handler)(c)

	assert.NoError(t, err)
	assert.Equal(t, true, beforeRan)
}

func TestRateLimiterMemoryStore_Allow(t *testing.T) {
	var inMemoryStore = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 3, ExpiresIn: 2 * time.Second})
	testCases := []struct {
		id      string
		allowed bool
	}{
		{"127.0.0.1", true},  // 0 ms
		{"127.0.0.1", true},  // 220 ms burst #2
		{"127.0.0.1", true},  // 440 ms burst #3
		{"127.0.0.1", false}, // 660 ms block
		{"127.0.0.1", false}, // 880 ms block
		{"127.0.0.1", true},  // 1100 ms next second #1
		{"127.0.0.2", true},  // 1320 ms allow other ip
		{"127.0.0.1", false}, // 1540 ms no burst
		{"127.0.0.1", false}, // 1760 ms no burst
		{"127.0.0.1", false}, // 1980 ms no burst
		{"127.0.0.1", true},  // 2200 ms no burst
		{"127.0.0.1", false}, // 2420 ms no burst
		{"127.0.0.1", false}, // 2640 ms no burst
		{"127.0.0.1", false}, // 2860 ms no burst
		{"127.0.0.1", true},  // 3080 ms no burst
		{"127.0.0.1", false}, // 3300 ms no burst
		{"127.0.0.1", false}, // 3520 ms no burst
		{"127.0.0.1", false}, // 3740 ms no burst
		{"127.0.0.1", false}, // 3960 ms no burst
		{"127.0.0.1", true},  // 4180 ms no burst
		{"127.0.0.1", false}, // 4400 ms no burst
		{"127.0.0.1", false}, // 4620 ms no burst
		{"127.0.0.1", false}, // 4840 ms no burst
		{"127.0.0.1", true},  // 5060 ms no burst
	}

	for i, tc := range testCases {
		t.Logf("Running testcase #%d => %v", i, time.Duration(i)*220*time.Millisecond)

		inMemoryStore.timeNow = func() time.Time {
			return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC).Add(time.Duration(i) * 220 * time.Millisecond)
		}
		allowed, _ := inMemoryStore.Allow(tc.id)
		assert.Equal(t, tc.allowed, allowed)
	}
}

func TestRateLimiterMemoryStore_cleanupStaleVisitors(t *testing.T) {
	var inMemoryStore = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 3})
	inMemoryStore.visitors = map[string]*Visitor{
		"A": {
			Limiter:  rate.NewLimiter(1, 3),
			lastSeen: time.Now(),
		},
		"B": {
			Limiter:  rate.NewLimiter(1, 3),
			lastSeen: time.Now().Add(-1 * time.Minute),
		},
		"C": {
			Limiter:  rate.NewLimiter(1, 3),
			lastSeen: time.Now().Add(-5 * time.Minute),
		},
		"D": {
			Limiter:  rate.NewLimiter(1, 3),
			lastSeen: time.Now().Add(-10 * time.Minute),
		},
	}

	inMemoryStore.Allow("D")
	inMemoryStore.cleanupStaleVisitors()

	var exists bool

	_, exists = inMemoryStore.visitors["A"]
	assert.Equal(t, true, exists)

	_, exists = inMemoryStore.visitors["B"]
	assert.Equal(t, true, exists)

	_, exists = inMemoryStore.visitors["C"]
	assert.Equal(t, false, exists)

	_, exists = inMemoryStore.visitors["D"]
	assert.Equal(t, true, exists)
}

func TestNewRateLimiterMemoryStore(t *testing.T) {
	testCases := []struct {
		rate              float64
		burst             int
		expiresIn         time.Duration
		expectedExpiresIn time.Duration
	}{
		{1, 3, 5 * time.Second, 5 * time.Second},
		{2, 4, 0, 3 * time.Minute},
		{1, 5, 10 * time.Minute, 10 * time.Minute},
		{3, 7, 0, 3 * time.Minute},
	}

	for _, tc := range testCases {
		store := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: tc.rate, Burst: tc.burst, ExpiresIn: tc.expiresIn})
		assert.Equal(t, tc.rate, store.rate)
		assert.Equal(t, tc.burst, store.burst)
		assert.Equal(t, tc.expectedExpiresIn, store.expiresIn)
	}
}

func generateAddressList(count int) []string {
	addrs := make([]string, count)
	for i := 0; i < count; i++ {
		addrs[i] = randomString(15)
	}

	return addrs
}

func run(wg *sync.WaitGroup, store RateLimiterStore, addrs []string, max int, b *testing.B) {
	for i := 0; i < b.N; i++ {
		store.Allow(addrs[rand.Intn(max)])
	}
	wg.Done()
}

func benchmarkStore(store RateLimiterStore, parallel int, max int, b *testing.B) {
	addrs := generateAddressList(max)

	wg := &sync.WaitGroup{}
	for i := 0; i < parallel; i++ {
		wg.Add(1)

		go run(wg, store, addrs, max, b)
	}
	wg.Wait()
}

const (
	testExpiresIn = 1000 * time.Millisecond
)

func BenchmarkRateLimiterMemoryStore_1000(b *testing.B) {
	var store = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 100, Burst: 200, ExpiresIn: testExpiresIn})
	benchmarkStore(store, 10, 1000, b)
}

func BenchmarkRateLimiterMemoryStore_10000(b *testing.B) {
	var store = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 100, Burst: 200, ExpiresIn: testExpiresIn})
	benchmarkStore(store, 10, 10000, b)
}

func BenchmarkRateLimiterMemoryStore_100000(b *testing.B) {
	var store = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 100, Burst: 200, ExpiresIn: testExpiresIn})
	benchmarkStore(store, 10, 100000, b)
}

func BenchmarkRateLimiterMemoryStore_conc100_10000(b *testing.B) {
	var store = NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 100, Burst: 200, ExpiresIn: testExpiresIn})
	benchmarkStore(store, 100, 10000, b)
}
