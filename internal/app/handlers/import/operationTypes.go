package handlers

import (
	"context"
	"fmt"
	"gar_converter/internal/config"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportOperationTypes(ctx *context.Context, config *config.Config, batch []interface{}) {
	// Create Postgres repository
	pgRepo, err := repository.NewPostgresRepository(*ctx,
		config.Database.DSN)
	if err != nil {
		fmt.Printf("Error creating Postgres repository: %v\n", err)
		return
	}
	defer pgRepo.Close()

	fmt.Printf("Start import %s...\n", GetBatchTypeStr(batch[0]))
	fmt.Println("Total operation types rows to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.OperationType); ok {
			pgxBatch.Queue(`
		insert into operation_types (id, name, is_active, update_date, start_date, end_date)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO UPDATE SET
		name=$2, is_active=$3, update_date=$4, start_date=$5, end_date=$6`,
				ao.ID,
				ao.Name,
				ao.IsActive,
				ao.UpdateDate,
				ao.StartDate,
				ao.EndDate,
			)
		} else {
			continue
		}
	}

	br := pgRepo.Pool().SendBatch(*ctx, &pgxBatch)
	defer br.Close()

	for range batch {
		if _, err := br.Exec(); err != nil {
			fmt.Printf("Batch operations types insert error %s\n", err)
			continue
		}
	}

	fmt.Printf("%s batch import completed... \n", GetBatchTypeStr(batch[0]))
}
