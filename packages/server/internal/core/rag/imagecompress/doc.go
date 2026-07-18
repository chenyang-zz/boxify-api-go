// Package imagecompress 提供图片压缩、缩放和 MIME 推断能力。
//
// 该包只处理图片字节，不调用模型、不访问存储。默认策略会在图片超过目标体积时解码、
// 约束最长边、贴白底并按 JPEG 质量逐级重编码；无法解码时返回原图，避免压缩失败阻断上层流程。
package imagecompress
