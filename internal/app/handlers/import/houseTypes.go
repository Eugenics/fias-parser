package handlers

import (
	"context"
	"fmt"
	"gar_converter/internal/config"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportHouseTypes(ctx *context.Context, config *config.Config, batch []interface{}) {
	// Create Postgres repository
	pgRepo, err := repository.NewPostgresRepository(*ctx,
		config.Database.DSN)
	if err != nil {
		fmt.Printf("Error creating Postgres repository: %v\n", err)
		return
	}
	defer pgRepo.Close()

	fmt.Printf("Start import %s...\n", GetBatchTypeStr(batch[0]))
	fmt.Println("Total params rows to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.HouseType); ok {
			pgxBatch.Queue(`
        insert into house_types
			(id, name, short_name, "desc", is_active, update_date, start_date, end_date)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (id) DO UPDATE SET
			name=$2,
			short_name=$3,
			"desc"=$4,
			is_active=$5,
			update_date=$6,
			start_date=$7,
			end_date=$8`,
				ao.ID,
				ao.Name,
				ao.ShortName,
				ao.Desc,
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
			fmt.Printf("Batch insert error %s\n", err)
			continue
		}
	}

	fmt.Printf("%s batch import completed... \n", GetBatchTypeStr(batch[0]))
}
