// Package search 提供通用 RAG 混合检索流程。
//
// 该包负责向量召回、BM25 召回、分数归一化、权重融合、可选重排和结果整形。
// 业务过滤、索引字段含义和来源元数据解码均由调用方通过 FilterBuilder、RequestOption 和 SourceDecoder 注入。
package search
