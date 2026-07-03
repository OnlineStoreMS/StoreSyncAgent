package kdzs

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Session keeps the main login token and per-platform trade sessions.
type Session struct {
	mu              sync.RWMutex
	client          *Client
	userID          string
	mobile          string
	accountID       string
	accountName     string
	accountRole     string
	platform        map[string]*PlatformSession
}

type PlatformSession struct {
	Host    string
	Token   string
	Cookies []*http.Cookie
	Referer string
}

func NewSession(client *Client) *Session {
	return &Session{
		client:   client,
		platform: make(map[string]*PlatformSession),
	}
}

func (s *Session) EnsureLogin(ctx context.Context, mobile, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client.Token() != "" {
		return nil
	}
	data, err := s.client.LoginWithPassword(ctx, mobile, password)
	if err != nil {
		return err
	}
	s.userID = data.UserID
	s.mobile = data.Mobile
	return nil
}

func (s *Session) SwitchAccount(ctx context.Context, accountID, accountName, accountRole, mobile, password string) error {
	s.mu.Lock()
	s.client.SetToken("")
	s.userID = ""
	s.mobile = ""
	s.accountID = accountID
	s.accountName = accountName
	s.accountRole = accountRole
	s.platform = make(map[string]*PlatformSession)
	s.mu.Unlock()
	if err := s.EnsureLogin(ctx, mobile, password); err != nil {
		return err
	}
	return nil
}

func (s *Session) AccountID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountID
}

func (s *Session) AccountName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountName
}

func (s *Session) AccountRole() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountRole
}

func (s *Session) Mobile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mobile
}

func (s *Session) UserID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userID
}

func (s *Session) PlatformSession(ctx context.Context, platform string) (*PlatformSession, error) {
	platform = strings.ToUpper(platform)
	s.mu.RLock()
	if ps, ok := s.platform[platform]; ok {
		s.mu.RUnlock()
		return ps, nil
	}
	s.mu.RUnlock()

	ps, err := s.openPlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.platform[platform] = ps
	s.mu.Unlock()
	return ps, nil
}

func (s *Session) openPlatformSession(ctx context.Context, platform string) (*PlatformSession, error) {
	if s.client.Token() == "" {
		return nil, fmt.Errorf("not logged in")
	}
	userID := s.UserID()
	if userID == "" {
		return nil, fmt.Errorf("missing user id")
	}

	path := fmt.Sprintf("?userId=%s#/allPack/", userID)
	redirectURL, err := s.client.GetRedirectURL(ctx, platform, path)
	if err != nil {
		return nil, fmt.Errorf("get redirect url: %w", err)
	}

	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{
		Timeout:       30 * time.Second,
		Jar:           jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil },
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("qnquerystring", s.client.Token())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open platform session: %w", err)
	}
	resp.Body.Close()

	finalURL := resp.Request.URL.String()
	parsed, _ := url.Parse(finalURL)
	host := parsed.Host
	if host == "" {
		if u, err := url.Parse(redirectURL); err == nil {
			host = u.Host
		}
	}
	if host == "" {
		host = PlatformHost[platform]
	}

	u, _ := url.Parse("https://" + host)
	cookies := jar.Cookies(u)
	token := s.client.Token()
	cookieName := fmt.Sprintf("rsid_%s-%s", platform, userID)
	for _, c := range cookies {
		if c.Name == cookieName {
			token = c.Value
			break
		}
	}

	return &PlatformSession{
		Host:    host,
		Token:   token,
		Cookies: cookies,
		Referer: finalURL,
	}, nil
}

func (ps *PlatformSession) CookieHeader() string {
	parts := make([]string, 0, len(ps.Cookies))
	for _, c := range ps.Cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}
