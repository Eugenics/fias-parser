package app

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	handlers "gar_converter/internal/app/handlers/import"
	"gar_converter/internal/config"
	"gar_converter/internal/domain"
	"gar_converter/internal/repository"
)

var excludedFileNames = []string{
	"AS_ADDR_OBJ_DIVISION", "AS_APARTMENTS", "AS_APARTMENTS_PARAMS", "AS_CARPLACES",
	"AS_CARPLACES_PARAMS", "AS_CHANGE_HISTORY", "AS_HOUSES", "AS_HOUSES_PARAMS",
	"AS_MUN_HIERARCHY", "AS_REESTR_OBJECTS", "AS_ROOMS", "AS_ROOMS_PARAMS",
	"AS_STEADS", "AS_STEADS_PARAMS",
}

func ReadXMLFile(ctx context.Context, path string, cfg *config.Config, repo *repository.PostgresRepository) error {
	for _, excluded := range excludedFileNames {
		if strings.Contains(path, excluded) {
			return nil
		}
	}
	fmt.Printf("Reading XML file: %s\n", path)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return Handle(ctx, file, cfg.Importer.BatchSize, repo)
}

func Handle(ctx context.Context, input io.Reader, batchSize int, repo *repository.PostgresRepository) error {
	decoder := xml.NewDecoder(input)
	batch := make([]interface{}, 0, batchSize)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		decoded, err := decodeElement(decoder, &start)
		if err != nil {
			return fmt.Errorf("decode %s: %w", start.Name.Local, err)
		}
		if decoded == nil {
			continue
		}
		batch = append(batch, decoded)
		if len(batch) >= batchSize {
			if err := processBatch(ctx, repo, batch); err != nil {
				return err
			}
			batch = make([]interface{}, 0, batchSize)
		}
	}
	if len(batch) > 0 {
		return processBatch(ctx, repo, batch)
	}
	return nil
}

func decodeElement(decoder *xml.Decoder, start *xml.StartElement) (interface{}, error) {
	var object domain.DecoderInterface
	switch start.Name.Local {
	case "OBJECT":
		object = &domain.AddressObject{}
	case "PARAM":
		object = &domain.Param{}
	case "HOUSETYPE":
		object = &domain.HouseType{}
	case "ADDRESSOBJECTTYPE":
		object = &domain.AddressObjectType{}
	case "APARTMENTTYPE":
		object = &domain.ApartmentType{}
	case "NDOCKIND":
		object = &domain.NdocKind{}
	case "NDOCTYPE":
		object = &domain.NdocType{}
	case "OBJECTLEVEL":
		object = &domain.ObjectLevel{}
	case "OPERATIONTYPE":
		object = &domain.OperationType{}
	case "PARAMTYPE":
		object = &domain.ParamType{}
	case "ROOMTYPE":
		object = &domain.RoomType{}
	case "ITEM":
		object = &domain.AdmHierarchyItem{}
	default:
		return nil, nil
	}
	return object.Decode(decoder, start)
}

func processBatch(ctx context.Context, repo *repository.PostgresRepository, batch []interface{}) error {
	if len(batch) == 0 {
		return nil
	}
	fmt.Printf("Processing batch of %d\n", len(batch))
	switch batch[0].(type) {
	case *domain.AddressObject:
		return handlers.ImportAddressObjects(ctx, repo, batch)
	case *domain.Param:
		return handlers.ImportParams(ctx, repo, batch)
	case *domain.HouseType:
		return handlers.ImportHouseTypes(ctx, repo, batch)
	case *domain.AddressObjectType:
		return handlers.ImportAddressObjectsTypes(ctx, repo, batch)
	case *domain.ApartmentType:
		return handlers.ImportApartmentTypes(ctx, repo, batch)
	case *domain.NdocKind:
		return handlers.ImportNdocKind(ctx, repo, batch)
	case *domain.NdocType:
		return handlers.ImportNdocTypes(ctx, repo, batch)
	case *domain.ObjectLevel:
		return handlers.ImportObjectLevels(ctx, repo, batch)
	case *domain.OperationType:
		return handlers.ImportOperationTypes(ctx, repo, batch)
	case *domain.ParamType:
		return handlers.ImportParamTypes(ctx, repo, batch)
	case *domain.RoomType:
		return handlers.ImportRoomTypes(ctx, repo, batch)
	case *domain.AdmHierarchyItem:
		return handlers.ImportAdmHierarchyItems(ctx, repo, batch)
	default:
		return fmt.Errorf("unsupported batch type %T", batch[0])
	}
}
