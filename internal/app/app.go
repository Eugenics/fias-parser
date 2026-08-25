package app

import (
	"fmt"
	config "gar_converter/internal/config"
	"sync"
)

type Cwg struct {
	wg    sync.WaitGroup
	count int
}

func Run(xmlFilesPath string) error {
	fmt.Println("Starting GAR converter...")
	fmt.Println("Loading structured data from configuration...")
	data := GetStucturedData(xmlFilesPath)

	// Load configuration
	config, err := config.Load("./configs/config.yaml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	var workers = config.Importer.Workers

	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for _, region := range data {
		for _, xmlFile := range region.XmlFiles {
			sem <- struct{}{}
			wg.Add(1)
			go func(b XMLFile) {
				defer wg.Done()
				defer func() { <-sem }()
				ReadXmlFile(xmlFile.Path)
			}(xmlFile)
		}
	}

	wg.Wait()
	return nil
}

// UniqueSlice accepts any slice where elements are comparable
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
