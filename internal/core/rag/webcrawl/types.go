package webcrawl

import (
	"context"
	"net"
	"net/http"
)

type Input struct {
	URL string
}

type Output struct {
	Title   string
	Content string
	URL     string
}

type Page struct {
	URL  string
	HTML []byte
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Extractor interface {
	Extract(ctx context.Context, page Page) (*Output, error)
}

type URLGuard interface {
	Validate(ctx context.Context, rawURL string) error
}

type Resolver interface {
	LookupIP(ctx context.Context, host string) ([]net.IP, error)
}

type extractorFunc func(ctx context.Context, page Page) (*Output, error)

func (fn extractorFunc) Extract(ctx context.Context, page Page) (*Output, error) {
	return fn(ctx, page)
}
