/**
 * @Time   : 2026/6/28 14:54
 * @Author : chenyangzhao542@gmail.com
 * @File   : agent_task.go
 **/

package models

import (
	"time"

	"github.com/google/uuid"
)

// 触发类型
type TriggerType string

const (
	TriggerDaily    TriggerType = "daily"    // 每天 HH:MM
	TriggerWeekly   TriggerType = "weekly"   // 每周某天 HH:MM
	TriggerInterval TriggerType = "interval" // 每隔 N 小时
)

type TaskRunType string

const (
	TaskRunNone    TaskRunType = ""
	TaskRunRunning TaskRunType = "running" // 运行中
	TaskRunDone    TaskRunType = "done"    // 完成
	TaskRunFailed  TaskRunType = "failed"  // 失败
)

type AgentTask struct {
	ID                   uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	UserID               uuid.UUID  `gorm:"column:user_id;type:uuid;not null;index"`
	User                 User       `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Name                 string     `gorm:"column:name;size:128;not null"`                        // 任务名
	Instruction          string     `gorm:"column:instruction;type:text"`                         // 自然语言研究指令/主题
	KDIDs                StringList `gorm:"column:kd_ids;type:jsonb;default:'[]'"`                // 检索范围，空=默认
	TriggerType          string     `gorm:"column:trigger_type;size:16;not null;default:'daily'"` // 触发类型
	TriggerTime          string     `gorm:"column:trigger_time;size:8;"`                          // HH:MM
	TriggerWeekday       *int       `gorm:"column:trigger_weekday;"`                              // 触发星期几，周日0 周一1
	TriggerIntervalHours *int       `gorm:"column:trigger_interval_hours;"`                       // 触发间隔小时数
	Enabled              bool       `gorm:"column:enabled;not null;default:true;index"`           // 是否启用
	NotifyEnabled        bool       `gorm:"column:notify_enabled;not null;default:true"`          // 本任务跑完是否推送到用户的消息渠道（默认推）
	LastRunAt            *time.Time `gorm:"column:last_run_at;"`                                  // 最后运行时间
	LastStatus           string     `gorm:"column:last_status;size:16;"`                          // 最后状态
	NextRunAt            *time.Time `gorm:"column:next_run_at;index"`                             // 下次运行时间
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AgentTask) TableName() string {
	return "agent_tasks"
}
