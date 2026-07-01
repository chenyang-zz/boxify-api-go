package webcrawl

import (
	"net/http"
	"time"
)

const (
	defaultTimeout      = 20 * time.Second
	defaultMaxRedirects = 5
	defaultRetryCount   = 1
)

type Options struct {
	HTTPClient   HTTPClient
	Extractor    Extractor
	URLGuard     URLGuard
	Timeout      time.Duration
	MaxRedirects int
	RetryCount   int
}

type Option func(*Options)

func WithHTTPClient(client HTTPClient) Option {
	return func(opts *Options) {
		if client != nil {
			opts.HTTPClient = client
		}
	}
}

func WithExtractor(extractor Extractor) Option {
	return func(opts *Options) {
		if extractor != nil {
			opts.Extractor = extractor
		}
	}
}

func WithURLGuard(guard URLGuard) Option {
	return func(opts *Options) {
		if guard != nil {
			opts.URLGuard = guard
		}
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		if timeout > 0 {
			opts.Timeout = timeout
		}
	}
}

func WithMaxRedirects(maxRedirects int) Option {
	return func(opts *Options) {
		if maxRedirects >= 0 {
			opts.MaxRedirects = maxRedirects
		}
	}
}

func WithRetryCount(retryCount int) Option {
	return func(opts *Options) {
		if retryCount >= 0 {
			opts.RetryCount = retryCount
		}
	}
}

func defaultHTTPClient(timeout time.Duration, maxRedirects int, guard URLGuard) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return http.ErrUseLastResponse
			}
			if guard != nil {
				return guard.Validate(req.Context(), req.URL.String())
			}
			return nil
		},
	}
}
