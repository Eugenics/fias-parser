package app

import (
	"context"
	"fmt"

	"gar_converter/internal/config"
	"gar_converter/internal/repository"

	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, xmlFilesPath string, cfg *config.Config, repo *repository.PostgresRepository) error {
	fmt.Println("Starting GAR converter...")
	data, err := GetStructuredData(xmlFilesPath)
	if err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(cfg.Importer.Workers)
	for _, region := range data {
		for _, file := range region.XmlFiles {
			file := file
			group.Go(func() error {
				if err := ReadXMLFile(groupCtx, file.Path, cfg, repo); err != nil {
					return fmt.Errorf("import %s: %w", file.Path, err)
				}
				return nil
			})
		}
	}
	return group.Wait()
}

func UniqueSlice[T comparable](input []T) []T {
	seen := make(map[T]struct{})
	distinct := make([]T, 0, len(input))
	for _, val := range input {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			distinct = append(distinct, val)
		}
	}
	return distinct
}
