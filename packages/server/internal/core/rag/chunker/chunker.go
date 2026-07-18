package chunker

import (
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

// Chunker 按 token 上限把文本拆成父子两级 chunk。
//
// Chunker 初始化后可复用；它持有 tokenizer 实例，不在分块过程中修改公开配置。
type Chunker struct {
	Options
	tkm *tiktoken.Tiktoken
}

// NewChunker 创建带默认配置的 Chunker。
//
// opts 会按传入顺序覆盖默认值。TokenEncodingName 无法被 tiktoken 加载时会 panic，
// 这是初始化配置错误，调用方应在启动或测试阶段暴露。
func NewChunker(opts ...Option) *Chunker {
	chunker := &Chunker{
		Options: Options{
			ChildChunkTokens:  defaultChildChunkTokens,
			ParentChunkTokens: defaultParentChunkTokens,
			ChildOverlapRatio: defaultChildOverlapRatio,
			SentenceRegex:     defaultSentenceRegex,
			TokenEncodingName: defaultTokenEncodingName,
		},
	}
	for _, opt := range opts {
		opt(&chunker.Options)
	}
	tkm, err := tiktoken.GetEncoding(chunker.TokenEncodingName)
	if err != nil {
		panic(err)
	}

	chunker.tkm = tkm

	return chunker
}

// Chunk 将文本拆成父块和子块。
//
// 空白文本返回 nil。父块不使用重叠，子块使用 ChildOverlapRatio 保留局部上下文。
func (c *Chunker) Chunk(text string) []*Chunk {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// 先按句子构建父块，再在每个父块内部切出子块，保证子块不会跨父块。
	sentences := c.splitSentence(text)
	parentChunks := c.MergeToChunk(sentences, c.ParentChunkTokens, 0)

	chunks := make([]*Chunk, 0, len(parentChunks))
	for _, pc := range parentChunks {
		parent := &Chunk{
			Content: pc,
		}
		childSentences := c.splitSentence(pc)
		parent.Children = c.MergeToChunk(childSentences, c.ChildChunkTokens, c.ChildOverlapRatio)
		chunks = append(chunks, parent)
	}

	return chunks
}

// CountTokens 返回文本在当前 tokenizer 编码下的 token 数。
func (c *Chunker) CountTokens(text string) int {
	tokens := c.tkm.Encode(text, nil, nil)
	return len(tokens)
}

// MergeToChunk 按目标 token 上限合并句子。
//
// 单句超过 targetTokens 时会拆成长句子子块；overlapRatio 大于 0 时，flush 后保留当前块尾部句子作为下一块上下文。
func (c *Chunker) MergeToChunk(sentences []string, targetTokens int, overlapRatio float64) []string {
	chunks := make([]string, 0)
	cur := make([]string, 0)
	curTokens := 0
	for _, sentence := range sentences {
		sentenceTokens := c.CountTokens(sentence)

		// 单句超过目标上限时，先 flush 已累积内容，再按 token 窗口拆分长句。
		if sentenceTokens > targetTokens {
			if len(cur) != 0 {
				chunks = append(chunks, strings.Join(cur, ""))
				cur, curTokens = make([]string, 0), 0
			}

			subChunks := c.splitLongSentence(sentence, targetTokens, overlapRatio)
			chunks = append(chunks, subChunks...)
			continue
		}

		// 新句子会导致超限时，先输出当前块，再根据重叠比例保留尾部句子。
		if sentenceTokens+curTokens > targetTokens && len(cur) != 0 {
			chunks = append(chunks, strings.Join(cur, ""))
			if overlapRatio > 0.0 {
				keep := max(1, int(float64(len(cur))*overlapRatio))
				cur = cur[len(cur)-keep:]
				curTokens = c.CountTokens(strings.Join(cur, ""))
			} else {
				cur, curTokens = make([]string, 0), 0
			}
		}

		cur = append(cur, sentence)
		curTokens += sentenceTokens
	}

	if len(cur) != 0 {
		chunks = append(chunks, strings.Join(cur, ""))
	}

	return chunks
}

// splitSentence 按句子正则分割文本。
func (c *Chunker) splitSentence(text string) []string {
	text = strings.TrimSpace(text)
	matches := c.SentenceRegex.FindAllString(text, -1)
	if matches == nil {
		return []string{text}
	}

	chunks := make([]string, 0, len(matches))
	for _, match := range matches {
		chunks = append(chunks, strings.TrimSpace(match))
	}

	return chunks
}

// splitLongSentence 按 token 窗口分割超过目标上限的长句。
func (c *Chunker) splitLongSentence(sentence string, targetTokens int, overlapRatio float64) []string {
	if targetTokens <= 0 {
		return nil
	}
	tokens := c.tkm.Encode(sentence, nil, nil)
	tokenCount := len(tokens)

	if tokenCount <= targetTokens {
		return []string{sentence}
	}

	// overlapRatio 会减少每次推进步长，让相邻长句子子块保留一段 token 上下文。
	step := targetTokens
	if overlapRatio > 0 {
		overlap := max(1, int(float64(targetTokens)*overlapRatio))
		step = max(1, targetTokens-overlap)
	}
	var chunks []string
	for start := 0; start < tokenCount; start += step {
		end := min(start+targetTokens, tokenCount)
		chunks = append(chunks, c.tkm.Decode(tokens[start:end]))
		if end == tokenCount {
			break
		}
	}
	return chunks
}
