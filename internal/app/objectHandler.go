package app

import (
	"context"
	"encoding/xml"
	"fmt"
	handlers "gar_converter/internal/app/handlers/import"
	config "gar_converter/internal/config"
	"gar_converter/internal/domain"
	"io"
	"os"
	"sync"
)

func Handle(xmlFile *os.File) error {

	ctx := context.Background()

	decoder := xml.NewDecoder(xmlFile)
	if decoder == nil {
		return fmt.Errorf("failed to create XML decoder")
	}

	config, err := config.Load("./configs/config.yaml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	batchSize := config.Importer.BatchSize
	workers := config.Importer.Workers

	batch := make([]interface{}, 0, batchSize)

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch se := tok.(type) {
		case xml.StartElement:
			seLocalName := se.Name.Local
			var decodeObject interface{} = nil
			var err error = nil

			switch seLocalName {
			case "OBJECT":
				var object domain.DecoderInterface = &domain.AddressObject{}
				decodeObject, err = object.Decode(decoder, &se)
			case "PARAM":
				var object domain.DecoderInterface = &domain.Param{}
				decodeObject, err = object.Decode(decoder, &se)
			case "HOUSETYPE":
				var object domain.DecoderInterface = &domain.HouseType{}
				decodeObject, err = object.Decode(decoder, &se)
			case "ADDRESSOBJECTTYPE":
				var object domain.DecoderInterface = &domain.AddressObjectType{}
				decodeObject, err = object.Decode(decoder, &se)
			case "APARTMENTTYPE":
				var object domain.DecoderInterface = &domain.ApartmentType{}
				decodeObject, err = object.Decode(decoder, &se)
			case "NDOCKIND":
				var object domain.DecoderInterface = &domain.NdocKind{}
				decodeObject, err = object.Decode(decoder, &se)
			case "NDOCTYPE":
				var object domain.DecoderInterface = &domain.NdocType{}
				decodeObject, err = object.Decode(decoder, &se)
			case "OBJECTLEVEL":
				var object domain.DecoderInterface = &domain.ObjectLevel{}
				decodeObject, err = object.Decode(decoder, &se)
			case "OPERATIONTYPE":
				var object domain.DecoderInterface = &domain.OperationType{}
				decodeObject, err = object.Decode(decoder, &se)
			case "PARAMTYPE":
				var object domain.DecoderInterface = &domain.ParamType{}
				decodeObject, err = object.Decode(decoder, &se)
			case "ROOMTYPE":
				var object domain.DecoderInterface = &domain.RoomType{}
				decodeObject, err = object.Decode(decoder, &se)
			case "ITEM":
				var object domain.DecoderInterface = &domain.AdmHierarchyItem{}
				decodeObject, err = object.Decode(decoder, &se)
			}

			if err != nil {
				fmt.Printf("Error decoding %s: %v\n", se.Name.Local, err)
				continue
			}

			if decodeObject != nil {
				batch = append(batch, decodeObject)
			}

			// Process batch if it reaches the configured size
			if len(batch) >= batchSize {
				currentBatch := make([]interface{}, len(batch))
				copy(currentBatch, batch)
				batch = batch[:0] // Clear the batch
				proccessBatch(currentBatch, &sem, &wg, &ctx, config)
			}
		}
	}

	// Process remaining chunks trailing at EOF
	if len(batch) > 0 {
		proccessBatch(batch, &sem, &wg, &ctx, config)
	}

	wg.Wait()
	return nil
}

func proccessBatch(
	batch []interface{},
	sem *chan struct{},
	wg *sync.WaitGroup,
	ctx *context.Context,
	config *config.Config,
) {
	fmt.Printf("Processing batch of %d %s\n", len(batch), "")

	*sem <- struct{}{}
	wg.Add(1)
	go func(b []interface{}) {
		defer wg.Done()
		defer func() { <-*sem }()

		switch batch[0].(type) {
		case *domain.AddressObject:
			handlers.ImportAddressObjects(ctx, config, b)
		case *domain.Param:
			handlers.ImportParams(ctx, config, b)
		case *domain.HouseType:
			handlers.ImportHouseTypes(ctx, config, b)
		case *domain.AddressObjectType:
			handlers.ImportAddressObjectsTypes(ctx, config, b)
		case *domain.ApartmentType:
			handlers.ImportApartmentTypes(ctx, config, b)
		case *domain.NdocKind:
			handlers.ImportNdocKind(ctx, config, b)
		case *domain.NdocType:
			handlers.ImportNdocTypes(ctx, config, b)
		case *domain.ObjectLevel:
			handlers.ImportObjectLevels(ctx, config, b)
		case *domain.OperationType:
			handlers.ImportOperationTypes(ctx, config, b)
		case *domain.ParamType:
			handlers.ImportParamTypes(ctx, config, b)
		case *domain.RoomType:
			handlers.ImportRoomTypes(ctx, config, b)
		case *domain.AdmHierarchyItem:
			handlers.ImportAdmHierarchyItems(ctx, config, b)
		}
	}(batch)
}
