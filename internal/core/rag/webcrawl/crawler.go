package webcrawl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Crawler struct {
	Options
}

func NewCrawler(opts ...Option) *Crawler {
	crawler := &Crawler{
		Options: Options{
			Timeout:      defaultTimeout,
			MaxRedirects: defaultMaxRedirects,
			RetryCount:   defaultRetryCount,
			Extractor:    NewHTMLExtractor(),
			URLGuard:     NewURLGuard(),
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&crawler.Options)
		}
	}
	if crawler.HTTPClient == nil {
		crawler.HTTPClient = defaultHTTPClient(crawler.Timeout, crawler.MaxRedirects, crawler.URLGuard)
	}
	return crawler
}

// Fetch 获取网页标题和内容
func (c *Crawler) Fetch(ctx context.Context, input Input) (*Output, error) {
	if c == nil || c.HTTPClient == nil {
		return nil, errors.New("rag web crawler http client is nil")
	}
	if c.URLGuard == nil {
		return nil, errors.New("rag web crawler url guard is nil")
	}
	if c.Extractor == nil {
		return nil, errors.New("rag web crawler extractor is nil")
	}
	rawURL := strings.TrimSpace(input.URL)
	if err := c.URLGuard.Validate(ctx, rawURL); err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= c.RetryCount; attempt++ {
		page, err := c.fetchOnce(ctx, rawURL)
		if err == nil {
			return c.Extractor.Extract(ctx, page)
		}
		lastErr = err
	}
	return nil, lastErr
}

// fetchOnce 获取网页内容
func (c *Crawler) fetchOnce(ctx context.Context, rawURL string) (Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Page{}, err
	}
	applyBrowserHeaders(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Page{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusInternalServerError {
		return Page{}, fmt.Errorf("web page server error: %s", resp.Status)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return Page{}, fmt.Errorf("web page request error: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Page{}, err
	}

	// 获取最终的 URL
	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	return Page{URL: finalURL, HTML: body}, nil
}

// applyBrowserHeaders 设置浏览器请求头
func applyBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
}
