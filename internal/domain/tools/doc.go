// Package tools 管理领域层本地工具集。
//
// 本包只负责把业务可用的本地工具组织成 core/tool.Catalog。工具实现必须保持
// 依赖轻量，不能反向依赖 svc、repository、HTTP 或数据库。
package tools
