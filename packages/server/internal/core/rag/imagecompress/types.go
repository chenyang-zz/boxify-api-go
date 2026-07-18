package imagecompress

// Input 表示一次图片压缩请求。
//
// Data 是原始图片字节，FileExt 用于推断 MIME；FileExt 可以带点或不带点。
type Input struct {
	Data    []byte
	FileExt string
}

// Output 表示图片压缩结果。
//
// Data 是可继续传给模型的图片字节。Compressed 表示图片是否经过重新编码。
// OriginalBytes 和 OutputBytes 分别记录输入和输出字节数。
type Output struct {
	Data          []byte
	MIME          string
	Compressed    bool
	OriginalBytes int
	OutputBytes   int
}
