package imagecompress

const (
	defaultMaxEdge     = 1568
	defaultTargetBytes = 3 * 1024 * 1024
)

var defaultQualities = []int{85, 70, 55, 40}

// Options 定义 Compressor 的长期压缩配置。
//
// MaxEdge 约束最长边，TargetBytes 是目标输出体积，Qualities 按顺序用于 JPEG 质量降级尝试。
type Options struct {
	MaxEdge     int
	TargetBytes int
	Qualities   []int
}

// Option 修改 Compressor 的长期压缩配置。
type Option func(*Options)

// WithMaxEdge 设置图片最长边上限。
func WithMaxEdge(maxEdge int) Option {
	return func(opts *Options) {
		if maxEdge > 0 {
			opts.MaxEdge = maxEdge
		}
	}
}

// WithTargetBytes 设置目标输出字节数。
func WithTargetBytes(targetBytes int) Option {
	return func(opts *Options) {
		if targetBytes > 0 {
			opts.TargetBytes = targetBytes
		}
	}
}

// WithQualities 设置 JPEG 质量降级序列。
//
// 空切片或全部非法质量值会保留默认配置；合法质量范围是 1 到 100。
func WithQualities(qualities []int) Option {
	return func(opts *Options) {
		if len(qualities) == 0 {
			return
		}
		out := make([]int, 0, len(qualities))
		for _, quality := range qualities {
			if quality <= 0 || quality > 100 {
				continue
			}
			out = append(out, quality)
		}
		if len(out) != 0 {
			opts.Qualities = out
		}
	}
}
