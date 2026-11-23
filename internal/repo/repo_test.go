package repo

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"navbox/internal/model"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db, mock
}

func TestSiteRepoCount(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "sites"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := NewSiteRepo(db).Count(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplySiteFilterTagMatch(t *testing.T) {
	db, mock := newMockDB(t)
	tagIDs := []uuid.UUID{uuid.New(), uuid.New()}

	tests := []struct {
		name       string
		tagMatch   string
		wantHaving bool
	}{
		{name: "all", tagMatch: SiteTagMatchAll, wantHaving: true},
		{name: "empty defaults to all", tagMatch: "", wantHaving: true},
		{name: "any", tagMatch: SiteTagMatchAny, wantHaving: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sites []model.Site
			query := db.Session(&gorm.Session{DryRun: true}).Model(&model.Site{})
			stmt := applySiteFilter(query, SiteListFilter{TagIDs: tagIDs, TagMatch: tt.tagMatch}).Find(&sites).Statement
			sql := stmt.SQL.String()
			hasHaving := strings.Contains(sql, "HAVING COUNT(DISTINCT tag_id) =")

			require.Equal(t, tt.wantHaving, hasHaving)
		})
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTagRepoCount(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tags"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := NewTagRepo(db).Count(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIconRepoCount(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "icons"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	count, err := NewIconRepo(db).Count(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(4), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIconRepoListQueries(t *testing.T) {
	db, _ := newMockDB(t)

	var icons []model.Icon
	listStmt := db.Session(&gorm.Session{DryRun: true}).
		Order("created_at DESC, id DESC").
		Find(&icons).Statement
	require.Contains(t, listStmt.SQL.String(), `ORDER BY created_at DESC, id DESC`)

	var refs []IconSiteRefRow
	refStmt := db.Session(&gorm.Session{DryRun: true}).
		Model(&model.Site{}).
		Select("id, title, icon_value").
		Where("icon_value LIKE ? OR icon_value LIKE ?", "/uploads/%", "uploads/%").
		Order("sort_order ASC, created_at ASC").
		Find(&refs).Statement
	require.Contains(t, refStmt.SQL.String(), `FROM "sites"`)
	require.Contains(t, refStmt.SQL.String(), `icon_value LIKE $1 OR icon_value LIKE $2`)
}

func TestApplyAuditFilterParentheses(t *testing.T) {
	db, _ := newMockDB(t)
	var logs []model.AuditLog
	query := db.Session(&gorm.Session{DryRun: true}).Model(&model.AuditLog{})
	stmt := applyAuditFilter(query, AuditFilter{
		ActorType: "admin",
		Query:     "github",
	}).Find(&logs).Statement
	sql := stmt.SQL.String()
	require.Contains(t, sql, "actor_type = $1")
	require.Contains(t, sql, "action ILIKE")
	require.Contains(t, sql, "(actor_name ILIKE")
}
