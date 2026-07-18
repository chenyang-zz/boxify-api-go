// Package classifier 提供面向 RAG 内容的轻量标签分类能力。
//
// 该包只依赖调用方注入的文本模型接口，不访问数据库、不绑定业务标签体系。
// 默认提示词来自 rag/prompt 模板；通过 WithPrompt 传入的提示词会被视为最终文本，不再做模板渲染。
package classifier
