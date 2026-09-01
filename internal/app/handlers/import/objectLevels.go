package handlers

import (
	"context"
	"errors"
	"fmt"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportObjectLevels(ctx context.Context, pgRepo *repository.PostgresRepository, batch []interface{}) error {
	fmt.Printf("Start import %s...\n", GetBatchTypeStr(batch[0]))
	fmt.Println("Total object levels rows to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.ObjectLevel); ok {
			pgxBatch.Queue(`
		insert into object_levels (level, name, start_date, end_date, update_date, is_active)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (level) DO UPDATE SET
		name=$2, start_date=$3, end_date=$4, update_date=$5, is_active=$6`,
				ao.Level,
				ao.Name,
				ao.StartDate,
				ao.EndDate,
				ao.UpdateDate,
				ao.IsActive,
			)
		} else {
			continue
		}
	}

	br := pgRepo.Pool().SendBatch(ctx, &pgxBatch)
	var batchErr error

	for range batch {
		if _, err := br.Exec(); err != nil {
			batchErr = errors.Join(batchErr, err)
			fmt.Printf("Batch object levels insert error %s\n", err)
			continue
		}
	}

	fmt.Printf("%s batch import completed... \n", GetBatchTypeStr(batch[0]))
	return errors.Join(batchErr, br.Close())
}
