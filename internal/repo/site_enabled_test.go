package repo

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"navbox/internal/model"
)

func TestSiteVisibilityFilter(t *testing.T) {
	for _, view := range []string{"", SiteViewFavorite, SiteViewUncategorized} {
		for _, public := range []bool{true, false} {
			t.Run(view+"/"+map[bool]string{true: "public", false: "admin"}[public], func(t *testing.T) {
				db, _ := newMockDB(t)
				var sites []model.Site
				stmt := applySiteFilter(db.Session(&gorm.Session{DryRun: true}).Model(&model.Site{}), SiteListFilter{
					EnabledOnly: public, View: view, Search: "example", TagIDs: []uuid.UUID{uuid.New()},
				}).Find(&sites).Statement
				require.NoError(t, stmt.Error)
				if public {
					require.Contains(t, stmt.SQL.String(), "sites.is_enabled = $1")
					require.Equal(t, true, stmt.Vars[0])
				} else {
					require.NotContains(t, stmt.SQL.String(), "sites.is_enabled")
				}
			})
		}
	}
}

func TestSiteCreateEnabledDefaultAndExplicitFalse(t *testing.T) {
	for _, requested := range []*bool{nil, siteEnabledPointer(true), siteEnabledPointer(false)} {
		t.Run(map[bool]string{true: "enabled", false: "disabled"}[requested == nil || *requested], func(t *testing.T) {
			db, mock := newMockDB(t)
			id := uuid.New()
			want := requested == nil || *requested
			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "sites"`).WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), "Test", "", "https://example.com", "", "new_window", "text", "N", "", false, want, false, 0,
			).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
			mock.ExpectExec(`DELETE FROM "site_tags"`).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
			site := model.Site{Title: "Test", DefaultURL: "https://example.com", IconValue: "N", IsEnabled: requested}
			require.NoError(t, NewSiteRepo(db).CreateWithTags(context.Background(), &site, nil))
			require.Equal(t, want, site.Enabled())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSiteUpdatePreservesOmittedEnabled(t *testing.T) {
	for _, requested := range []*bool{nil, siteEnabledPointer(false), siteEnabledPointer(true)} {
		db, mock := newMockDB(t)
		id := uuid.New()
		mock.ExpectBegin()
		statement := `UPDATE "sites" SET "background_color"=$1,"default_url"=$2,"description"=$3,"icon_type"=$4,"icon_value"=$5,`
		args := []any{"", "https://example.com", "", "", ""}
		if requested != nil {
			statement += `"is_enabled"=$6,"is_favorite"=$7,"lan_url"=$8,"only_name"=$9,"open_method"=$10,"sort_order"=$11,"title"=$12,"updated_at"=$13 WHERE id = $14`
			args = append(args, *requested)
		} else {
			statement += `"is_favorite"=$6,"lan_url"=$7,"only_name"=$8,"open_method"=$9,"sort_order"=$10,"title"=$11,"updated_at"=$12 WHERE id = $13`
		}
		args = append(args, false, "", false, "", 0, "Test", sqlmock.AnyArg(), id)
		expect := mock.ExpectExec(regexp.QuoteMeta(statement))
		// sqlmock 的参数类型为 driver.Value，通过转换保留精确的 false 断言。
		values := make([]driver.Value, len(args))
		for i, arg := range args {
			values[i] = arg
		}
		expect.WithArgs(values...).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM "site_tags"`).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		require.NoError(t, NewSiteRepo(db).UpdateWithTags(context.Background(), &model.Site{
			Base: model.Base{ID: id}, Title: "Test", DefaultURL: "https://example.com", IsEnabled: requested,
		}, nil))
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestSiteImportKeepsDisabled(t *testing.T) {
	db, mock := newMockDB(t)
	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "sites"`).WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), "Test", "", "https://example.com", "", "new_window", "text", "", "", false, false, false, 0, id,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
	mock.ExpectCommit()
	require.NoError(t, NewImportExportRepo(db).Import(context.Background(), ImportBatch{Sites: []model.Site{{
		Base: model.Base{ID: id}, Title: "Test", DefaultURL: "https://example.com", IsEnabled: siteEnabledPointer(false),
	}}}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func siteEnabledPointer(value bool) *bool { return &value }
