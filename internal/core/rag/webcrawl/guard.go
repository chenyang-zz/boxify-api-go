package webcrawl

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
)

// URLGuardOptions 定义默认 URLGuard 的配置。
type URLGuardOptions struct {
	Resolver Resolver
}

// URLGuardOption 修改默认 URLGuard 的配置。
type URLGuardOption func(*URLGuardOptions)

type defaultURLGuard struct {
	resolver Resolver
}

type netResolver struct{}

// NewURLGuard 创建默认 URL 安全校验器。
//
// 默认校验器只允许 http/https，并拒绝解析到本地、内网、链路本地、多播或未指定地址的 host。
func NewURLGuard(opts ...URLGuardOption) URLGuard {
	options := URLGuardOptions{Resolver: netResolver{}}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return &defaultURLGuard{resolver: options.Resolver}
}

// WithResolver 设置 URLGuard 使用的域名解析器。
func WithResolver(resolver Resolver) URLGuardOption {
	return func(opts *URLGuardOptions) {
		if resolver != nil {
			opts.Resolver = resolver
		}
	}
}

// LookupIP 使用 net.DefaultResolver 解析 host。
func (netResolver) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}

// Validate 校验 URL 是否可以安全抓取。
func (g *defaultURLGuard) Validate(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("unsupported url scheme")
	}
	host := parsed.Hostname()
	if host == "" {
		return errors.New("url host is required")
	}

	ips := []net.IP{}
	// host 可以直接是 IP，也可以是域名；域名必须解析后逐个检查。
	if ip := net.ParseIP(host); ip != nil {
		ips = append(ips, ip)
	} else {
		resolved, err := g.resolver.LookupIP(ctx, host)
		if err != nil {
			return err
		}
		ips = append(ips, resolved...)
	}
	if len(ips) == 0 {
		return errors.New("url host has no ip")
	}
	for _, ip := range ips {
		if !isSafeIP(ip) {
			return errors.New("unsafe url host ip")
		}
	}
	return nil
}

// isSafeIP reports whether IP 属于可公网访问的普通单播地址。
func isSafeIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
