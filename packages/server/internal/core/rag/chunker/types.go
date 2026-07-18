package chunker

// Chunk 表示一个父文本块及其下属的子文本块。
//
// Content 保存父块原文，Children 保存同一父块内按更小 token 上限切出的子块。
type Chunk struct {
	Content  string
	Children []string
}
