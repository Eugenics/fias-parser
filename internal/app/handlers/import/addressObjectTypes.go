package handlers

import (
	"context"
	"errors"
	"fmt"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportAddressObjectsTypes(ctx context.Context, pgRepo *repository.PostgresRepository, batch []interface{}) error {
	fmt.Printf("Start import %s...\n", GetBatchTypeStr(batch[0]))
	fmt.Println("Total params rows to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.AddressObjectType); ok {
			pgxBatch.Queue(`
        insert into address_object_types
			(id, level, name, short_name, "desc", update_date, start_date, end_date, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (id) DO UPDATE SET
			level=$2,
			name=$3,
			short_name=$4,
			"desc"=$5,
			update_date=$6,
			start_date=$7,
			end_date=$8,
			is_active=$9`,
				ao.ID,
				ao.Level,
				ao.Name,
				ao.ShortName,
				ao.Desc,
				ao.UpdateDate,
				ao.StartDate,
				ao.EndDate,
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
			fmt.Printf("Batch insert error %s\n", err)
			continue
		}
	}

	fmt.Printf("%s batch import completed... \n", GetBatchTypeStr(batch[0]))
	return errors.Join(batchErr, br.Close())
}
