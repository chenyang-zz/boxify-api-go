// Package webcrawl 提供安全网页抓取和正文提取能力。
//
// 该包默认包含 SSRF 防护、浏览器风格请求头、有限重试和 HTML 标题/正文抽取。
// 调用方可以替换 HTTPClient、Extractor、URLGuard 或 Resolver，以接入更强的网络和解析实现。
package webcrawl
