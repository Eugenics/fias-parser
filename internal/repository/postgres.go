package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"gar_converter/internal/db"
	"gar_converter/internal/download"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(ctx context.Context, dsn string) (*PostgresRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping connection pool: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

func (r *PostgresRepository) Close() {
	r.pool.Close()
}

func (r *PostgresRepository) Pool() *pgxpool.Pool {
	return r.pool
}

var dateLayouts = []string{"2006-01-02", "2006.01.02", "02.01.2006"}

var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// SaveVersionInfo upserts the given FIAS version info into version_info.
// Unparseable dates and missing URLs are stored as NULL.
func (r *PostgresRepository) SaveVersionInfo(ctx context.Context, info *download.LastInfo, status, fileType string) error {
	q := db.New(r.pool)

	params := db.UpsertVersionInfoParams{
		VersionID:   info.VersionID,
		TextVersion: info.TextVersion,
		Status:      pgtype.Text{String: status, Valid: true},
		FileType:    pgtype.Text{String: fileType, Valid: fileType != ""}.String,
	}

	if url := firstNonEmpty(info.FiasCompleteXmlUrl, info.GarXMLFullURL); url != "" {
		params.GarXmlFullUrl = pgtype.Text{String: url, Valid: true}
	}
	if url := firstNonEmpty(info.FiasDeltaXmlUrl, info.GarXMLDeltaURL); url != "" {
		params.GarXmlDeltaUrl = pgtype.Text{String: url, Valid: true}
	}
	if t, ok := parseTime(info.ExpDate, timeLayouts); ok {
		params.ExpDate = pgtype.Timestamp{Time: t, Valid: true}
	}
	if t, ok := parseTime(info.Date, dateLayouts); ok {
		params.Date = pgtype.Date{Time: t, Valid: true}
	}

	return q.UpsertVersionInfo(ctx, params)
}

// IsVersionImported reports whether the given version has already been
// imported into the DB (status = 'imported').
func (r *PostgresRepository) IsVersionImported(ctx context.Context, versionID int64) (bool, error) {
	return db.New(r.pool).VersionImported(ctx, versionID)
}

// ExtractionBlocker returns an already processed version that prevents a new
// extraction. The selected version takes priority over other pending versions.
func (r *PostgresRepository) ExtractionBlocker(ctx context.Context, versionID int64, fileType string) (int64, string, bool, error) {
	row, err := db.New(r.pool).ExtractionBlocker(ctx, versionID, fileType)
	if err == pgx.ErrNoRows {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, err
	}
	return row.VersionID, row.Status.String, true, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseTime(value string, layouts []string) (time.Time, bool) {
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
