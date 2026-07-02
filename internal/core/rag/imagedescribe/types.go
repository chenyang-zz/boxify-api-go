package imagedescribe

import (
	"context"

	"github.com/boxify/api-go/internal/core/rag/imagecompress"
)

// VisionClient 定义图片描述所需的最小视觉模型能力。
//
// Describe 接收最终提示词、base64 图片、MIME 和最大 token 数，返回模型原文。
type VisionClient interface {
	Describe(ctx context.Context, prompt string, imageBase64 string, mime string, maxTokens int64) (string, error)
}

// Compressor 定义图片描述前的压缩能力。
type Compressor interface {
	Compress(input imagecompress.Input) (*imagecompress.Output, error)
}

// Input 表示一次图片描述请求。
//
// Data 是原始图片字节，FileExt 用于压缩阶段推断 MIME。
type Input struct {
	Data    []byte
	FileExt string
}

// Description 表示模型生成的结构化图片描述。
//
// Description 是主要描述文本，OCRText 是识别出的文字，Objects 是对象列表，Scene 是简短场景标签。
type Description struct {
	Description string   `json:"description"`
	OCRText     string   `json:"ocr_text"`
	Objects     []string `json:"objects"`
	Scene       string   `json:"scene"`
}
