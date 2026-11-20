package dto

import (
	"encoding/json"
	"time"
)

type CronJobResp struct {
	ID              string      `json:"id"`
	JobKey          string      `json:"job_key"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	CronExpr        string      `json:"cron_expr"`
	DefaultSchedule string      `json:"default_schedule"`
	Enabled         bool        `json:"enabled"`
	TimeoutSec      int         `json:"timeout_sec"`
	ConfigJSON      string      `json:"config_json"`
	LastRunAt       *time.Time  `json:"last_run_at,omitempty"`
	LastStatus      string      `json:"last_status"`
	LastDurationMs  int64       `json:"last_duration_ms"`
	LastSummary     string      `json:"last_summary,omitempty"`
	NextRunAt       *time.Time  `json:"next_run_at,omitempty"`
	NextRuns        []time.Time `json:"next_runs"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type CronJobListResp struct {
	Items []CronJobResp `json:"items"`
}

type CronJobRunResp struct {
	ID           string     `json:"id"`
	JobID        string     `json:"job_id"`
	TriggerType  string     `json:"trigger_type"`
	TriggeredBy  string     `json:"triggered_by"`
	Status       string     `json:"status"`
	AffectedRows int64      `json:"affected_rows"`
	OutputLog    string     `json:"output_log"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

type CronJobRunListResp struct {
	Items []CronJobRunResp `json:"items"`
}

type PreviewCronReq struct {
	Schedule string `json:"schedule"`
	CronExpr string `json:"cron_expr"`
}

type PreviewCronResp struct {
	NextRuns []time.Time `json:"next_runs"`
}

type UpdateCronJobReq struct {
	CronExpr   string          `json:"cron_expr"`
	Enabled    bool            `json:"enabled"`
	TimeoutSec *int            `json:"timeout_sec"`
	Config     json.RawMessage `json:"config"`
}

type CronJobSecretsResp struct {
	APIKey string `json:"api_key"`
}

type TestSafeKeepReq struct {
	APIBaseURL    string  `json:"api_base_url"`
	APIKey        *string `json:"api_key"`
	SkipTLSVerify *bool   `json:"skip_tls_verify"`
}

type SafeKeepUpstreamResp struct {
	UpstreamID   string `json:"upstream_id"`
	UpstreamName string `json:"upstream_name"`
	UpstreamType string `json:"upstream_type"`
	Enabled      bool   `json:"enabled"`
	IsDefault    bool   `json:"is_default"`
	Priority     int    `json:"priority"`
}

type TestSafeKeepResp struct {
	OK        bool                   `json:"ok"`
	Upstreams []SafeKeepUpstreamResp `json:"upstreams"`
}
