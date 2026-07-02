package webcrawl

import (
	"context"
	"net"
	"net/http"
)

// Input 表示一次网页抓取请求。
type Input struct {
	URL string
}

// Output 表示网页抓取和正文提取结果。
type Output struct {
	Title   string
	Content string
	URL     string
}

// Page 表示 HTTP 抓取后的原始网页。
type Page struct {
	URL  string
	HTML []byte
}

// HTTPClient 定义 Crawler 需要的最小 HTTP 能力。
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Extractor 定义网页正文提取能力。
type Extractor interface {
	Extract(ctx context.Context, page Page) (*Output, error)
}

// URLGuard 定义 URL 安全校验能力。
//
// Validate 应在请求发起前执行，也会用于默认 HTTP client 的重定向校验。
type URLGuard interface {
	Validate(ctx context.Context, rawURL string) error
}

// Resolver 定义 URLGuard 解析域名所需的最小能力。
type Resolver interface {
	LookupIP(ctx context.Context, host string) ([]net.IP, error)
}

type extractorFunc func(ctx context.Context, page Page) (*Output, error)

// Extract 调用函数形式的正文提取器。
func (fn extractorFunc) Extract(ctx context.Context, page Page) (*Output, error) {
	return fn(ctx, page)
}
