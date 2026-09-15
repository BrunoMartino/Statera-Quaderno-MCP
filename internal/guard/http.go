package guard

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const httpTimeout = 10 * time.Second

// StoreCaller performs allowlisted HTTPS to one WP_BASE_URL after Policy checks.
type StoreCaller struct {
	BaseURL     string
	Auth        interface{ Apply(*http.Request) }
	Policy      *AccessPolicy
	HTTP        *http.Client
	Logger      *slog.Logger
	StoreID     string
	Environment string
}

func NewStoreCaller(baseURL string, auth interface{ Apply(*http.Request) }, policy *AccessPolicy) *StoreCaller {
	parsed, _ := url.Parse(baseURL)
	allowedHost := ""
	if parsed != nil {
		allowedHost = parsed.Host
	}
	client := &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Host != allowedHost {
				return errors.New("redirect host not allowed")
			}
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			if auth != nil {
				auth.Apply(req)
			}
			return nil
		},
	}
	return &StoreCaller{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Auth:    auth,
		Policy:  policy,
		HTTP:    client,
	}
}

func (s *StoreCaller) Do(ctx context.Context, method, path string, body []byte, contentType string) ([]byte, error) {
	return s.DoWithHeaders(ctx, method, path, body, contentType, nil)
}

// DoWithHeaders is Do plus extra request headers (the observability shared secret).
// Header values are never logged.
func (s *StoreCaller) DoWithHeaders(ctx context.Context, method, path string, body []byte, contentType string, headers map[string]string) ([]byte, error) {
	if s == nil || s.Auth == nil {
		return nil, NewError(AuthMissing)
	}
	if s.Policy != nil {
		if err := s.Policy.AssertRequest(method, path); err != nil {
			return nil, err
		}
	}
	attempts := 1
	if method == http.MethodGet {
		attempts = 2
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		b, status, err := s.doOnce(ctx, method, path, body, contentType, headers)
		if err != nil {
			lastErr = err
			if method != http.MethodGet {
				return nil, err
			}
			continue
		}
		if status >= 500 && method == http.MethodGet && i == 0 {
			lastErr = fmtStatus(status, b)
			continue
		}
		if status >= 400 {
			return nil, fmtStatus(status, b)
		}
		s.logCall(method, path, status)
		return b, nil
	}
	return nil, lastErr
}

func (s *StoreCaller) doOnce(ctx context.Context, method, path string, body []byte, contentType string, headers map[string]string) ([]byte, int, error) {
	full := s.BaseURL + path
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, full, rdr)
	if err != nil {
		return nil, 0, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	s.Auth.Apply(req)
	if u := req.URL.Query(); u.Get("consumer_key") != "" || u.Get("consumer_secret") != "" {
		return nil, 0, NewError(FieldForbidden)
	}
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

func (s *StoreCaller) logCall(method, path string, status int) {
	if s.Logger == nil {
		return
	}
	s.Logger.Info("store http",
		"method", method,
		"path", path,
		"status", status,
		"store", s.StoreID,
		"env", s.Environment,
	)
}
