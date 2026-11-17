package httpcache

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCacheBudgetAndHeaders(t *testing.T) {
	calls := 0
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("x-key") != "secret" {
			t.Errorf("missing auth header")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("body")), Header: http.Header{}}, nil
	})
	now := time.Date(2026, 11, 5, 6, 0, 0, 0, time.UTC)
	var slept time.Duration
	c := &Client{
		HTTP:   &http.Client{Transport: rt},
		Dir:    t.TempDir(),
		MinGap: time.Second,
		Budget: 2,
		Header: http.Header{"x-key": {"secret"}},
		Now:    func() time.Time { return now },
		Sleep:  func(_ context.Context, d time.Duration) error { slept += d; return nil },
	}
	ctx := context.Background()
	b, cached, err := c.Get(ctx, Request{URL: "http://x/a", Key: "a"})
	if err != nil || cached || string(b) != "body" {
		t.Fatalf("first: %q %v %v", b, cached, err)
	}
	if _, cached, _ = c.Get(ctx, Request{URL: "http://x/a", Key: "a"}); !cached {
		t.Fatal("second call should be cached")
	}
	if _, _, err = c.Get(ctx, Request{URL: "http://x/b", Key: "b", TTL: -1}); err != nil {
		t.Fatal(err)
	}
	if slept != time.Second {
		t.Fatalf("want one MinGap sleep, got %v", slept)
	}
	if _, _, err = c.Get(ctx, Request{URL: "http://x/c", Key: "c"}); !errors.Is(err, ErrBudget) {
		t.Fatalf("want budget error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("network calls: %d", calls)
	}
}

func TestHTTPErrorNotCached(t *testing.T) {
	n := 0
	rt := rtFunc(func(*http.Request) (*http.Response, error) {
		n++
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("slow down")), Header: http.Header{}}, nil
	})
	c := &Client{HTTP: &http.Client{Transport: rt}, Dir: t.TempDir(), Now: func() time.Time { return time.Unix(0, 0) }}
	for i := 0; i < 2; i++ {
		if _, _, err := c.Get(context.Background(), Request{URL: "http://x", Key: "k"}); err == nil {
			t.Fatal("want error")
		}
	}
	if n != 2 {
		t.Fatalf("error response must not be cached, calls=%d", n)
	}
}

func TestOffline(t *testing.T) {
	calls := 0
	rt := rtFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("body")), Header: http.Header{}}, nil
	})
	now := time.Date(2026, 11, 5, 6, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	on := &Client{HTTP: &http.Client{Transport: rt}, Dir: dir, Now: func() time.Time { return now }}
	ctx := context.Background()
	if _, _, err := on.Get(ctx, Request{URL: "http://x/a", Key: "a", TTL: time.Hour}); err != nil {
		t.Fatal(err)
	}
	off := &Client{HTTP: &http.Client{Transport: rt}, Dir: dir, Offline: true, Now: func() time.Time { return now.Add(48 * time.Hour) }}
	// Expired by TTL but cached: offline serves it.
	if b, cached, err := off.Get(ctx, Request{URL: "http://x/a", Key: "a", TTL: time.Hour}); err != nil || !cached || string(b) != "body" {
		t.Fatalf("offline hit: %q %v %v", b, cached, err)
	}
	if _, _, err := off.Get(ctx, Request{URL: "http://x/b", Key: "b"}); !errors.Is(err, ErrOffline) {
		t.Fatalf("want offline error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("network calls: %d", calls)
	}
}
