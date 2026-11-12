// Package httpcache is a polite GET client: on-disk cache, minimum spacing between
// requests and an optional request budget. Free tiers (API-HOCKEY 100 req/day) and
// public sites (swehockey) are the reason; re-runs must cost nothing.
package httpcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrBudget is returned when the request budget is used up.
var ErrBudget = errors.New("request budget exhausted")

// ErrOffline is returned by an Offline client for a URL that is not cached.
var ErrOffline = errors.New("not in cache (offline)")

// Client fetches URLs with caching.
type Client struct {
	HTTP      *http.Client
	Dir       string        // cache directory; empty disables caching
	MinGap    time.Duration // minimum spacing between network requests
	Budget    int           // max network requests in this process; 0 = unlimited
	Offline   bool          // serve any cached copy regardless of TTL, never touch the network
	UserAgent string
	Header    http.Header // extra headers (auth); never part of the cache key
	Now       func() time.Time
	Sleep     func(context.Context, time.Duration) error

	mu      sync.Mutex
	last    time.Time
	used    int
	OnReply func(*http.Response) // e.g. read rate-limit headers
}

// Request describes one GET. Key is the cache key; use the URL without secrets.
type Request struct {
	URL string
	Key string
	TTL time.Duration // 0 = cache forever (finished seasons); <0 = never cache
}

// Used returns network requests made so far.
func (c *Client) Used() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.used
}

// Get returns the body and whether it came from cache.
func (c *Client) Get(ctx context.Context, r Request) ([]byte, bool, error) {
	path := c.path(r)
	if c.Offline {
		if path == "" {
			return nil, false, ErrOffline
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, false, fmt.Errorf("%s: %w", r.Key, ErrOffline)
		}
		return b, true, nil
	}
	if path != "" && r.TTL >= 0 {
		if st, err := os.Stat(path); err == nil && (r.TTL == 0 || c.now().Sub(st.ModTime()) < r.TTL) {
			b, err := os.ReadFile(path)
			if err == nil {
				return b, true, nil
			}
		}
	}
	if err := c.wait(ctx); err != nil {
		return nil, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.URL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("build request: %w", err)
	}
	for k, vs := range c.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("get %s: %w", r.Key, err)
	}
	defer resp.Body.Close()
	if c.OnReply != nil {
		c.OnReply(resp)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", r.Key, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("get %s: http %d: %.200s", r.Key, resp.StatusCode, body)
	}
	if path != "" && r.TTL >= 0 {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, false, fmt.Errorf("cache dir: %w", err)
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, body, 0o644); err != nil {
			return nil, false, fmt.Errorf("cache write: %w", err)
		}
		if err := os.Rename(tmp, path); err != nil {
			return nil, false, fmt.Errorf("cache rename: %w", err)
		}
	}
	return body, false, nil
}

func (c *Client) wait(ctx context.Context) error {
	c.mu.Lock()
	if c.Budget > 0 && c.used >= c.Budget {
		c.mu.Unlock()
		return ErrBudget
	}
	c.used++
	gap := c.MinGap - c.now().Sub(c.last)
	c.mu.Unlock()
	if gap > 0 {
		sleep := c.Sleep
		if sleep == nil {
			sleep = sleepCtx
		}
		if err := sleep(ctx, gap); err != nil {
			return err
		}
	}
	c.mu.Lock()
	c.last = c.now()
	c.mu.Unlock()
	slog.Debug("http get", "n", c.used)
	return nil
}

func (c *Client) path(r Request) string {
	if c.Dir == "" {
		return ""
	}
	key := r.Key
	if key == "" {
		key = r.URL
	}
	h := sha256.Sum256([]byte(key))
	name := hex.EncodeToString(h[:])
	return filepath.Join(c.Dir, name[:2], name+".body")
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
