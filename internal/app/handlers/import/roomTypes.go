package handlers

import (
	"context"
	"fmt"
	"gar_converter/internal/config"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportRoomTypes(ctx *context.Context, config *config.Config, batch []interface{}) {
	// Create Postgres repository
	pgRepo, err := repository.NewPostgresRepository(*ctx,
		config.Database.DSN)
	if err != nil {
		fmt.Printf("Error creating Postgres repository: %v\n", err)
		return
	}
	defer pgRepo.Close()

	fmt.Printf("Start import %s...\n", GetBatchTypeStr(batch[0]))
	fmt.Println("Total room types rows to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.RoomType); ok {
			pgxBatch.Queue(`
		insert into room_types (id, name, "desc", is_active, update_date, start_date, end_date)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (id) DO UPDATE SET
		name=$2,"desc"=$3, is_active=$4, update_date=$5, start_date=$6, end_date=$7`,
				ao.ID,
				ao.Name,
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
			fmt.Printf("Batch room types insert error %s\n", err)
			continue
		}
	}

	fmt.Printf("%s batch import completed... \n", GetBatchTypeStr(batch[0]))
}
