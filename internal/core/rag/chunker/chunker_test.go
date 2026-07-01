package chunker

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewChunkerAppliesOptions(t *testing.T) {
	// 验证 NewChunker 使用默认 tokenizer，并且所有 ChunkOption 都能覆盖默认配置。
	sentenceRegex := regexp.MustCompile(`[A-Z]+`)

	c := NewChunker(
		WithChildChunkTokens(3),
		WithParentChunkTokens(5),
		WithChildOverlapRatio(0.25),
		WithSentenceRegex(sentenceRegex),
		WithTokenEncodingName(defaultTokenEncodingName),
	)

	if c.tkm == nil {
		t.Fatal("tokenizer is nil")
	}
	if c.CountTokens("hello") == 0 {
		t.Fatal("CountTokens returned 0 for non-empty text")
	}
	if c.ChildChunkTokens != 3 {
		t.Fatalf("ChildChunkTokens = %d, want 3", c.ChildChunkTokens)
	}
	if c.ParentChunkTokens != 5 {
		t.Fatalf("ParentChunkTokens = %d, want 5", c.ParentChunkTokens)
	}
	if c.ChildOverlapRatio != 0.25 {
		t.Fatalf("ChildOverlapRatio = %v, want 0.25", c.ChildOverlapRatio)
	}
	if c.SentenceRegex != sentenceRegex {
		t.Fatalf("SentenceRegex = %v, want custom regex", c.SentenceRegex)
	}
	if c.TokenEncodingName != defaultTokenEncodingName {
		t.Fatalf("TokenEncodingName = %q, want %q", c.TokenEncodingName, defaultTokenEncodingName)
	}
}

func TestChunkReturnsNilForBlankText(t *testing.T) {
	// 验证空白文本不会生成无意义 chunk。
	got := NewChunker().Chunk(" \n\t ")
	if got != nil {
		t.Fatalf("Chunk() = %#v, want nil", got)
	}
}

func TestChunkBuildsParentAndChildChunks(t *testing.T) {
	// 验证 Chunk 会先合并 parent chunk，再为每个 parent 生成 child chunk。
	base := NewChunker()
	parentLimit := base.CountTokens("A.") + base.CountTokens("B.")
	childLimit := base.CountTokens("A.")
	c := NewChunker(
		WithParentChunkTokens(parentLimit),
		WithChildChunkTokens(childLimit),
		WithChildOverlapRatio(0),
	)

	got := c.Chunk("A. B. C.")
	if len(got) != 2 {
		t.Fatalf("chunk count = %d, want 2: %#v", len(got), got)
	}
	if got[0].Content != "A.B." {
		t.Fatalf("first parent content = %q, want %q", got[0].Content, "A.B.")
	}
	if !slices.Equal(got[0].Children, []string{"A.", "B."}) {
		t.Fatalf("first parent children = %#v, want %#v", got[0].Children, []string{"A.", "B."})
	}
	if got[1].Content != "C." {
		t.Fatalf("second parent content = %q, want %q", got[1].Content, "C.")
	}
	if !slices.Equal(got[1].Children, []string{"C."}) {
		t.Fatalf("second parent children = %#v, want %#v", got[1].Children, []string{"C."})
	}
}

func TestMergeToChunkFlushesWhenTokenLimitExceeded(t *testing.T) {
	// 验证新句子加入后超过 token 上限时，会先输出当前 chunk。
	c := NewChunker()
	sentences := []string{"A.", "B.", "C."}
	targetTokens := c.CountTokens("A.") + c.CountTokens("B.")

	got := c.MergeToChunk(sentences, targetTokens, 0)
	want := []string{"A.B.", "C."}

	if !slices.Equal(got, want) {
		t.Fatalf("MergeToChunk() = %#v, want %#v", got, want)
	}
}

func TestMergeToChunkKeepsSentenceOverlap(t *testing.T) {
	// 验证 overlapRatio 大于 0 时，下一个 chunk 会保留上一段末尾句子。
	c := NewChunker()
	sentences := []string{"A.", "B.", "C."}
	targetTokens := c.CountTokens("A.") + c.CountTokens("B.")

	got := c.MergeToChunk(sentences, targetTokens, 0.5)
	want := []string{"A.B.", "B.C."}

	if !slices.Equal(got, want) {
		t.Fatalf("MergeToChunk() = %#v, want %#v", got, want)
	}
}

func TestMergeToChunkSplitsLongSentence(t *testing.T) {
	// 验证单句超过目标 token 数时，会按 token 窗口拆成多个子块。
	c := NewChunker()
	sentence := strings.Repeat("long sentence ", 80)
	targetTokens := max(1, c.CountTokens(sentence)/4)

	got := c.MergeToChunk([]string{sentence}, targetTokens, 0)
	if len(got) <= 1 {
		t.Fatalf("long sentence chunks = %#v, want more than one chunk", got)
	}
	for i, chunk := range got {
		if tokens := c.CountTokens(chunk); tokens > targetTokens {
			t.Fatalf("chunk %d token count = %d, want <= %d", i, tokens, targetTokens)
		}
	}
}

func TestNoopSearcherReturnsEmptyResult(t *testing.T) {
	// 验证默认空检索器不返回 citation，也不产生错误。
	got, err := NoopSearcher{}.Search(context.Background(), uuid.New(), "query", []string{"kb-1"}, 3)
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("Search() = %#v, want nil", got)
	}
}
