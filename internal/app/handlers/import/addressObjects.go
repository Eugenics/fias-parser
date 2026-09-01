package handlers

import (
	"context"
	"errors"
	"fmt"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportAddressObjects(ctx context.Context, pgRepo *repository.PostgresRepository, batch []interface{}) error {
	fmt.Println("Start import address objects...")
	fmt.Println("Total address objects to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.AddressObject); ok {
			pgxBatch.Queue(`
        insert into address_objects
			(id, object_id, object_guid, change_id, name, type_name, level, oper_type_id, prev_id,
             next_id, update_date, start_date, end_date, is_actual, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
        ON CONFLICT (id) DO UPDATE SET
			object_id=$2,
			object_guid=$3,
			change_id=$4,
			name=$5,
			type_name=$6,
			level=$7,
			oper_type_id=$8,
			prev_id=$9,
			next_id=$10,
			update_date=$11,
			start_date=$12,
			end_date=$13,
			is_actual=$14,
			is_active=$15`,
				ao.Id,
				ao.ObjectId,
				ao.ObjectGuid,
				ao.ChangeId,
				ao.Name,
				ao.TypeName,
				ao.Level,
				ao.OperTypeId,
				ao.PrevId,
				ao.NextId,
				ao.UpdateDate,
				ao.StartDate,
				ao.EndDate,
				(ao.IsActual != 0),
				(ao.IsActive != 0),
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
			fmt.Printf("Batch insert address objects error: %s\n", err)
			continue
		}
	}

	fmt.Println("Address objects batch import completed...")
	return errors.Join(batchErr, br.Close())
}
