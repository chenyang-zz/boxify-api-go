// Package chunker 提供面向 RAG 入库的文本分块能力。
//
// 该包按句子切分文本，再组装父块和子块；父块提供较完整上下文，子块用于更细粒度召回。
// 分块策略通过 Options 和 Option 注入，调用方可以调整 token 上限、重叠比例、句子正则和 tokenizer 编码。
package chunker
