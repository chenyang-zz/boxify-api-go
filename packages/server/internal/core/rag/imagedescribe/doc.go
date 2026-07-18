// Package imagedescribe 提供图片结构化描述能力。
//
// 该包先通过可注入的 Compressor 控制图片体积，再调用可注入的 VisionClient，
// 最后把模型输出规整为 Description。包内不依赖具体大模型 SDK、数据库或业务对象。
package imagedescribe
