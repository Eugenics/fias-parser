package handlers

import (
	"context"
	"fmt"
	"gar_converter/internal/config"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"

	"github.com/jackc/pgx/v5"
)

func ImportAdmHierarchyItems(ctx *context.Context, config *config.Config, batch []interface{}) {
	// Create Postgres repository
	pgRepo, err := repository.NewPostgresRepository(*ctx,
		config.Database.DSN)
	if err != nil {
		fmt.Printf("Error creating Postgres repository: %v\n", err)
		return
	}
	defer pgRepo.Close()

	fmt.Printf("Start import %s...\n", GetBatchTypeStr(batch[0]))
	fmt.Println("Total adm hierarchy items rows to import:", len(batch))

	pgxBatch := pgx.Batch{}

	for _, item := range batch {
		if ao, ok := item.(*domain.AdmHierarchyItem); ok {
			pgxBatch.Queue(`
       insert into adm_hierarchy 
	   		(id, object_id, parent_obj_id, change_id, region_code, prev_id, next_id, update_date,
                start_date, end_date, is_active, path)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT (id) DO UPDATE SET
			object_id=$2,
			parent_obj_id=$3,
			change_id=$4,
			region_code=$5,
			prev_id=$6,
			next_id=$7,
			update_date=$8,
            start_date=$9,
			end_date=$10,
			is_active=$11,
			path=$12`,
				ao.ID,
				ao.ObjectID,
				ao.ParentObjID,
				ao.ChangeID,
				ao.RegionCode,
				ao.PrevID,
				ao.NextID,
				ao.UpdateDate,
				ao.StartDate,
				ao.EndDate,
				(ao.IsActive != 0),
				ao.Path,
			)
		} else {
			continue
		}
	}

	br := pgRepo.Pool().SendBatch(*ctx, &pgxBatch)
	defer br.Close()

	for range batch {
		if _, err := br.Exec(); err != nil {
			fmt.Printf("Batch adm hierarchy items insert error %s\n", err)
			continue
		}
	}

	fmt.Printf("%s batch import completed... \n", GetBatchTypeStr(batch[0]))
}
