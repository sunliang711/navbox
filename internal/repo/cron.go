package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"navbox/internal/model"
)

type CronRepo interface {
	SeedDefaultJobs(ctx context.Context, jobs []model.CronJob) error
	ListJobs(ctx context.Context) ([]model.CronJob, error)
	GetJobByID(ctx context.Context, id uuid.UUID) (*model.CronJob, error)
	GetJobByKey(ctx context.Context, key string) (*model.CronJob, error)
	UpdateJob(ctx context.Context, id uuid.UUID, cronExpr string, enabled bool, timeoutSec int, configJSON string) (bool, error)
	UpdateJobNextRun(ctx context.Context, id uuid.UUID, nextRun *time.Time) error
	UpdateJobExecution(ctx context.Context, id uuid.UUID, status string, summary string, durationMs int64, runAt time.Time, nextRun *time.Time) error
	CreateRun(ctx context.Context, run *model.CronJobRun) error
	FinishRun(ctx context.Context, id uuid.UUID, status string, affectedRows int64, outputLog string, finishedAt time.Time) error
	ListRuns(ctx context.Context, jobID uuid.UUID, limit int) ([]model.CronJobRun, error)
	DeleteRunsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type cronRepo struct {
	db *gorm.DB
}

func NewCronRepo(db *gorm.DB) CronRepo {
	return &cronRepo{db: db}
}

func (r *cronRepo) SeedDefaultJobs(ctx context.Context, jobs []model.CronJob) error {
	for i := range jobs {
		job := jobs[i]
		if job.ID == uuid.Nil {
			job.ID = uuid.New()
		}
		if err := r.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "job_key"}},
				DoNothing: true,
			}).
			Create(&job).Error; err != nil {
			return fmt.Errorf("seed cron job %q: %w", job.JobKey, err)
		}
	}
	return nil
}

func (r *cronRepo) ListJobs(ctx context.Context) ([]model.CronJob, error) {
	var jobs []model.CronJob
	if err := r.db.WithContext(ctx).Order("job_key ASC").Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("list cron jobs: %w", err)
	}
	return jobs, nil
}

func (r *cronRepo) GetJobByID(ctx context.Context, id uuid.UUID) (*model.CronJob, error) {
	var job model.CronJob
	if err := r.db.WithContext(ctx).Take(&job, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get cron job: %w", err)
	}
	return &job, nil
}

func (r *cronRepo) GetJobByKey(ctx context.Context, key string) (*model.CronJob, error) {
	var job model.CronJob
	if err := r.db.WithContext(ctx).Take(&job, "job_key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get cron job by key: %w", err)
	}
	return &job, nil
}

func (r *cronRepo) UpdateJob(ctx context.Context, id uuid.UUID, cronExpr string, enabled bool, timeoutSec int, configJSON string) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&model.CronJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"cron_expr":   cronExpr,
			"enabled":     enabled,
			"timeout_sec": timeoutSec,
			"config_json": configJSON,
			"updated_at":  time.Now().UTC(),
		})
	if result.Error != nil {
		return false, fmt.Errorf("update cron job: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *cronRepo) UpdateJobNextRun(ctx context.Context, id uuid.UUID, nextRun *time.Time) error {
	if err := r.db.WithContext(ctx).
		Model(&model.CronJob{}).
		Where("id = ?", id).
		Update("next_run_at", nextRun).Error; err != nil {
		return fmt.Errorf("update cron job next run: %w", err)
	}
	return nil
}

func (r *cronRepo) UpdateJobExecution(ctx context.Context, id uuid.UUID, status string, summary string, durationMs int64, runAt time.Time, nextRun *time.Time) error {
	if err := r.db.WithContext(ctx).
		Model(&model.CronJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_run_at":      runAt,
			"last_status":      status,
			"last_duration_ms": durationMs,
			"last_summary":     summary,
			"next_run_at":      nextRun,
			"updated_at":       time.Now().UTC(),
		}).Error; err != nil {
		return fmt.Errorf("update cron job execution: %w", err)
	}
	return nil
}

func (r *cronRepo) CreateRun(ctx context.Context, run *model.CronJobRun) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return fmt.Errorf("create cron job run: %w", err)
	}
	return nil
}

func (r *cronRepo) FinishRun(ctx context.Context, id uuid.UUID, status string, affectedRows int64, outputLog string, finishedAt time.Time) error {
	if err := r.db.WithContext(ctx).
		Model(&model.CronJobRun{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        status,
			"affected_rows": affectedRows,
			"output_log":    outputLog,
			"finished_at":   finishedAt,
		}).Error; err != nil {
		return fmt.Errorf("finish cron job run: %w", err)
	}
	return nil
}

func (r *cronRepo) ListRuns(ctx context.Context, jobID uuid.UUID, limit int) ([]model.CronJobRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var runs []model.CronJobRun
	if err := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("started_at DESC").
		Limit(limit).
		Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("list cron job runs: %w", err)
	}
	return runs, nil
}

func (r *cronRepo) DeleteRunsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("started_at < ?", cutoff).Delete(&model.CronJobRun{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete cron job runs before %s: %w", cutoff.Format(time.RFC3339), result.Error)
	}
	return result.RowsAffected, nil
}
