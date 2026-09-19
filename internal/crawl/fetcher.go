package crawl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

const (
	maxPageBytes  = 5 << 20
	robotsTTL     = time.Hour
	requestTimout = 30 * time.Second
)

// ErrDisallowed means robots.txt forbids fetching the URL.
var ErrDisallowed = errors.New("disallowed by robots.txt")

// ErrAIInputRefused means the site's robots.txt carries a Content-Signal with
// ai-input=no: the publisher does not allow its content to be given to an AI
// model, which is exactly what summarizing does. It wraps ErrDisallowed.
var ErrAIInputRefused = fmt.Errorf("%w: Content-Signal ai-input=no", ErrDisallowed)

// aiInputRefused matches a Content-Signal line that opts out of AI input.
// https://contentsignals.org: ai-input covers retrieval, grounding and
// summarization; ai-train (which we never do) covers model training.
var aiInputRefused = regexp.MustCompile(`(?im)^\s*content-signal\s*:.*\bai-input\s*=\s*no\b`)

// Fetcher is the single door to outlet sites. It respects robots.txt and
// never sends two requests to the same host closer together than HostDelay.
type Fetcher struct {
	UserAgent string
	HostDelay time.Duration
	Client    *http.Client

	mu     sync.Mutex
	hosts  map[string]*hostState
	robots map[string]robotsEntry
}

type hostState struct {
	mu   sync.Mutex
	next time.Time
}

type robotsEntry struct {
	group     *robotstxt.Group
	noAIInput bool
	fetched   time.Time
}

func NewFetcher(userAgent string, hostDelay time.Duration) *Fetcher {
	return &Fetcher{
		UserAgent: userAgent,
		HostDelay: hostDelay,
		Client:    &http.Client{Timeout: requestTimout},
		hosts:     make(map[string]*hostState),
		robots:    make(map[string]robotsEntry),
	}
}

// Purpose says what a fetched page will be used for, because publishers
// allow different things for different uses.
type Purpose int

const (
	// ForAI: the content will be given to a model. Refused by a site whose
	// robots.txt says Content-Signal ai-input=no.
	ForAI Purpose = iota
	// ForIndex: only link metadata (headline, URL, time) is kept, the way a
	// search engine lists a page. Used for feeds and sitemaps of outlets in
	// headline-only coverage; nothing fetched this way may reach a model.
	ForIndex
)

// Get fetches a URL for AI use after checking robots.txt. The body is capped at 5 MB.
func (f *Fetcher) Get(ctx context.Context, rawURL string) ([]byte, *url.URL, error) {
	return f.GetFor(ctx, rawURL, ForAI)
}

// GetFor is Get with an explicit purpose.
func (f *Fetcher) GetFor(ctx context.Context, rawURL string, purpose Purpose) ([]byte, *url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, nil, fmt.Errorf("invalid url %q", rawURL)
	}
	allowed, err := f.allowed(ctx, u, purpose)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, fmt.Errorf("%w: %s", ErrDisallowed, rawURL)
	}
	body, final, err := f.get(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	return body, final, nil
}

func (f *Fetcher) allowed(ctx context.Context, u *url.URL, purpose Purpose) (bool, error) {
	origin := u.Scheme + "://" + u.Host

	f.mu.Lock()
	entry, ok := f.robots[origin]
	f.mu.Unlock()

	if !ok || time.Since(entry.fetched) > robotsTTL {
		robotsURL, _ := url.Parse(origin + "/robots.txt")
		body, status, err := f.do(ctx, robotsURL)
		if err != nil {
			return false, fmt.Errorf("fetch robots.txt for %s: %w", u.Host, err)
		}
		// FromStatusAndBytes applies the standard rules: 4xx means no
		// restrictions, 5xx means assume everything is disallowed.
		data, err := robotstxt.FromStatusAndBytes(status, body)
		if err != nil {
			return false, fmt.Errorf("parse robots.txt for %s: %w", u.Host, err)
		}
		entry = robotsEntry{
			group:     data.FindGroup(f.UserAgent),
			noAIInput: status == http.StatusOK && aiInputRefused.Match(body),
			fetched:   time.Now(),
		}
		f.mu.Lock()
		f.robots[origin] = entry
		f.mu.Unlock()
	}
	// robots.txt itself stays fetchable; everything else on the site does not.
	if entry.noAIInput && purpose == ForAI {
		return false, fmt.Errorf("%w (%s)", ErrAIInputRefused, u.Host)
	}
	return entry.group.Test(u.RequestURI()), nil
}

func (f *Fetcher) get(ctx context.Context, u *url.URL) ([]byte, *url.URL, error) {
	body, status, final, err := f.doFollow(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	if status != http.StatusOK {
		return nil, nil, fmt.Errorf("GET %s: status %d", u, status)
	}
	return body, final, nil
}

func (f *Fetcher) do(ctx context.Context, u *url.URL) ([]byte, int, error) {
	body, status, _, err := f.doFollow(ctx, u)
	return body, status, err
}

func (f *Fetcher) doFollow(ctx context.Context, u *url.URL) ([]byte, int, *url.URL, error) {
	if err := f.wait(ctx, u.Host); err != nil {
		return nil, 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("User-Agent", f.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en;q=0.5")

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
	if err != nil {
		return nil, 0, nil, err
	}
	return body, resp.StatusCode, resp.Request.URL, nil
}

// wait blocks until this host may be contacted again, then reserves the next slot.
func (f *Fetcher) wait(ctx context.Context, host string) error {
	f.mu.Lock()
	state, ok := f.hosts[host]
	if !ok {
		state = &hostState{}
		f.hosts[host] = state
	}
	f.mu.Unlock()

	state.mu.Lock()
	now := time.Now()
	at := state.next
	if at.Before(now) {
		at = now
	}
	state.next = at.Add(f.HostDelay)
	state.mu.Unlock()

	if delay := time.Until(at); delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
