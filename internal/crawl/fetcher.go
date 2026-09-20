package crawl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

const maxPageBytes = 5 << 20

var ErrDisallowed = errors.New("disallowed by robots.txt")

type HTTPError struct {
	Status     int
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d", e.Status) }

// Fetcher checks each redirect and shares a host clock across all requests.
type Fetcher struct {
	UserAgent string
	HostDelay time.Duration
	Client    *http.Client
	mu        sync.Mutex
	hosts     map[string]time.Time
	robots    map[string]robotsEntry
}
type robotsEntry struct {
	group   *robotstxt.Group
	fetched time.Time
}

func NewFetcher(agent string, delay time.Duration) *Fetcher {
	return &Fetcher{UserAgent: agent, HostDelay: delay, Client: &http.Client{Timeout: 30 * time.Second}, hosts: map[string]time.Time{}, robots: map[string]robotsEntry{}}
}
func (f *Fetcher) Get(ctx context.Context, raw string) ([]byte, *url.URL, error) {
	return f.follow(ctx, raw, false)
}
func (f *Fetcher) allowed(ctx context.Context, u *url.URL) error {
	origin := u.Scheme + "://" + u.Host
	f.mu.Lock()
	entry, ok := f.robots[origin]
	f.mu.Unlock()
	if !ok || time.Since(entry.fetched) > time.Hour {
		body, _, err := f.follow(ctx, origin+"/robots.txt", true)
		status := 200
		if err != nil {
			var he *HTTPError
			if !errors.As(err, &he) || he.Status == 429 || he.Status >= 500 {
				return err
			}
			status = he.Status
		}
		data, err := robotstxt.FromStatusAndBytes(status, body)
		if err != nil {
			return err
		}
		entry = robotsEntry{data.FindGroup(f.UserAgent), time.Now()}
		f.mu.Lock()
		f.robots[origin] = entry
		f.mu.Unlock()
	}
	if !entry.group.Test(u.RequestURI()) {
		return fmt.Errorf("%w: %s", ErrDisallowed, u)
	}
	return nil
}
func (f *Fetcher) follow(ctx context.Context, raw string, robots bool) ([]byte, *url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, nil, err
	}
	client := *f.Client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	for hop := 0; hop <= 5; hop++ {
		if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return nil, nil, fmt.Errorf("invalid URL %q", u)
		}
		if !robots {
			if err := f.allowed(ctx, u); err != nil {
				return nil, u, err
			}
		}
		if err := f.wait(ctx, u.Host); err != nil {
			return nil, u, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, u, err
		}
		req.Header.Set("User-Agent", f.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en;q=0.5")
		resp, err := client.Do(req)
		if err != nil {
			return nil, u, err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes+1))
		resp.Body.Close()
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			if hop == 5 {
				return nil, u, errors.New("too many redirects")
			}
			next, err := resp.Location()
			if err != nil {
				return nil, u, err
			}
			u = next
			continue
		}
		if resp.StatusCode != 200 {
			delay := time.Duration(0)
			if n, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
				delay = time.Duration(n) * time.Second
			} else if at, err := http.ParseTime(resp.Header.Get("Retry-After")); err == nil {
				delay = time.Until(at)
			}
			if resp.StatusCode == 429 {
				if delay < time.Minute {
					delay = time.Minute
				}
				f.mu.Lock()
				if at := time.Now().Add(delay); at.After(f.hosts[u.Host]) {
					f.hosts[u.Host] = at
				}
				f.mu.Unlock()
			}
			return nil, u, &HTTPError{resp.StatusCode, delay}
		}
		if readErr != nil {
			return nil, u, readErr
		}
		if len(body) > maxPageBytes {
			return nil, u, errors.New("response exceeds 5 MB")
		}
		return body, u, nil
	}
	return nil, u, errors.New("too many redirects")
}
func (f *Fetcher) wait(ctx context.Context, host string) error {
	for {
		f.mu.Lock()
		at := f.hosts[host]
		if !at.After(time.Now()) {
			f.hosts[host] = time.Now().Add(f.HostDelay)
			f.mu.Unlock()
			return nil
		}
		f.mu.Unlock()
		t := time.NewTimer(time.Until(at))
		select {
		case <-t.C:
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		}
	}
}
