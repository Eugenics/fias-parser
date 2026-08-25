package handlers

import (
	"context"
	"fmt"
	"gar_converter/internal/config"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportNdocTypes(ctx *context.Context, config *config.Config, batch []interface{}) {
	// Create Postgres repository
	pgRepo, err := repository.NewPostgresRepository(*ctx,
		config.Database.DSN)
	if err != nil {
		fmt.Printf("Error creating Postgres repository: %v\n", err)
		return
	}
	defer pgRepo.Close()

	fmt.Printf("Start import %s...\n", GetBatchTypeStr(batch[0]))
	fmt.Println("Total ndocTypes rows to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.NdocType); ok {
			pgxBatch.Queue(`
		insert into ndoc_types (id, name, start_date, end_date)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (id) DO UPDATE SET
			name=$2,
			start_date=$3,
			end_date=$4`,
				ao.ID,
				ao.Name,
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
			fmt.Printf("Batch ndoctype insert error %s\n", err)
			continue
		}
	}

	fmt.Printf("%s batch import completed... \n", GetBatchTypeStr(batch[0]))
}
