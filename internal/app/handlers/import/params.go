package handlers

import (
	"context"
	"fmt"
	"gar_converter/internal/config"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportParams(ctx *context.Context, config *config.Config, batch []interface{}) {
	// Create Postgres repository
	pgRepo, err := repository.NewPostgresRepository(*ctx,
		config.Database.DSN)
	if err != nil {
		fmt.Printf("Error creating Postgres repository: %v\n", err)
		return
	}
	defer pgRepo.Close()

	fmt.Println("Start import params...")
	fmt.Println("Total params rows to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.Param); ok {
			pgxBatch.Queue(`
        insert into params 
			(id, object_id, change_id, change_id_end, type_id, "value", update_date, start_date, end_date)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (id) DO UPDATE SET
			object_id=$2,
			change_id=$3,
			change_id_end=$4,
			type_id=$5,
			"value"=$6,
			update_date=$7,
			start_date=$8,
			end_date=$9`,
				ao.Id,
				ao.ObjectId,
				ao.ChangeId,
				ao.ChangeIdEnd,
				ao.TypeId,
				ao.Value,
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
			fmt.Printf("Batch insert error params %s\n", err)
			continue
		}
	}

	fmt.Println("Params batch import completed...")
}
