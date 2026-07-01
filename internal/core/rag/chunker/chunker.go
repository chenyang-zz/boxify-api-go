package chunker

import (
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

type Chunker struct {
	ChunkOptions
	tkm *tiktoken.Tiktoken
}

func NewChunker(opts ...ChunkOption) *Chunker {
	chunker := &Chunker{
		ChunkOptions: ChunkOptions{
			ChildChunkTokens:  defaultChildChunkTokens,
			ParentChunkTokens: defaultParentChunkTokens,
			ChildOverlapRatio: defaultChildOverlapRatio,
			SentenceRegex:     defaultSentenceRegex,
			TokenEncodingName: defaultTokenEncodingName,
		},
	}
	for _, opt := range opts {
		opt(&chunker.ChunkOptions)
	}
	tkm, err := tiktoken.GetEncoding(chunker.TokenEncodingName)
	if err != nil {
		panic(err)
	}

	chunker.tkm = tkm

	return chunker
}

// Chunk 文本分块 父子分块
func (c *Chunker) Chunk(text string) []*Chunk {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

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

// CountTokens 计算文本的 token 数
func (c *Chunker) CountTokens(text string) int {
	tokens := c.tkm.Encode(text, nil, nil)
	return len(tokens)
}

// MergeToChunk 合并成 chunk
func (c *Chunker) MergeToChunk(sentences []string, targetTokens int, overlapRatio float64) []string {
	chunks := make([]string, 0)
	cur := make([]string, 0)
	curTokens := 0
	for _, sentence := range sentences {
		sentenceTokens := c.CountTokens(sentence)

		// 如果当前句子的 token 数大于目标 token 数，则单独成 chunk
		if sentenceTokens > targetTokens {
			if len(cur) != 0 {
				chunks = append(chunks, strings.Join(cur, ""))
				cur, curTokens = make([]string, 0), 0
			}

			subChunks := c.splitLongSentence(sentence, targetTokens, overlapRatio)
			chunks = append(chunks, subChunks...)
			continue
		}

		// 如果当前句子加入后超过目标 token 数，则将当前 chunk 添加到 chunks 中，并重置 cur 和 curTokens
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

// splitSentence 按句子分割文本
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

// splitLongSentence 分割长句子
func (c *Chunker) splitLongSentence(sentence string, targetTokens int, overlapRatio float64) []string {
	if targetTokens <= 0 {
		return nil
	}
	tokens := c.tkm.Encode(sentence, nil, nil)
	tokenCount := len(tokens)

	if tokenCount <= targetTokens {
		return []string{sentence}
	}
	// 每次向前推进多少 token
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
