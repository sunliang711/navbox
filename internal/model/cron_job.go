package model

import (
	"time"

	"github.com/google/uuid"
)

// CronJob 描述内置定时任务的配置与最近执行状态。
type CronJob struct {
	Base
	JobKey         string     `json:"-" gorm:"column:job_key;type:varchar(100);not null;uniqueIndex"`
	Name           string     `json:"-" gorm:"column:name;type:varchar(200);not null"`
	Description    string     `json:"-" gorm:"column:description;type:text;not null;default:''"`
	CronExpr       string     `json:"-" gorm:"column:cron_expr;type:varchar(100);not null"`
	Enabled        bool       `json:"-" gorm:"column:enabled;not null;default:true"`
	TimeoutSec     int        `json:"-" gorm:"column:timeout_sec;not null;default:300"`
	ConfigJSON     string     `json:"-" gorm:"column:config_json;type:text;not null;default:'{}'"`
	LastRunAt      *time.Time `json:"-" gorm:"column:last_run_at"`
	LastStatus     string     `json:"-" gorm:"column:last_status;type:varchar(32);not null;default:'never'"`
	LastDurationMs int64      `json:"-" gorm:"column:last_duration_ms;not null;default:0"`
	LastSummary    string     `json:"-" gorm:"column:last_summary;type:text;not null;default:''"`
	NextRunAt      *time.Time `json:"-" gorm:"column:next_run_at"`
}

func (CronJob) TableName() string {
	return "cron_jobs"
}

// CronJobRun 记录定时任务的单次执行历史。
type CronJobRun struct {
	ID           uuid.UUID  `json:"-" gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	JobID        uuid.UUID  `json:"-" gorm:"column:job_id;type:uuid;not null;index:idx_cron_runs_job_started,priority:1"`
	TriggerType  string     `json:"-" gorm:"column:trigger_type;type:varchar(32);not null"`
	TriggeredBy  string     `json:"-" gorm:"column:triggered_by;type:varchar(128);not null;default:''"`
	Status       string     `json:"-" gorm:"column:status;type:varchar(32);not null"`
	AffectedRows int64      `json:"-" gorm:"column:affected_rows;not null;default:0"`
	OutputLog    string     `json:"-" gorm:"column:output_log;type:text;not null;default:''"`
	StartedAt    time.Time  `json:"-" gorm:"column:started_at;not null;index:idx_cron_runs_job_started,priority:2,sort:desc"`
	FinishedAt   *time.Time `json:"-" gorm:"column:finished_at"`
}

func (CronJobRun) TableName() string {
	return "cron_job_runs"
}
