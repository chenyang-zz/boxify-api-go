package webcrawl

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
)

type URLGuardOptions struct {
	Resolver Resolver
}

type URLGuardOption func(*URLGuardOptions)

type defaultURLGuard struct {
	resolver Resolver
}

type netResolver struct{}

func NewURLGuard(opts ...URLGuardOption) URLGuard {
	options := URLGuardOptions{Resolver: netResolver{}}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return &defaultURLGuard{resolver: options.Resolver}
}

func WithResolver(resolver Resolver) URLGuardOption {
	return func(opts *URLGuardOptions) {
		if resolver != nil {
			opts.Resolver = resolver
		}
	}
}

// LookupIP 使用net.DefaultResolver进行IP解析
func (netResolver) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}

// Validate 验证URL是否安全
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
	// 尝试解析IP
	if ip := net.ParseIP(host); ip != nil {
		ips = append(ips, ip)
	} else {
		// 解析域名
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

// isSafeIP 检查IP是否安全
func isSafeIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
