package handlers

import "gar_converter/internal/domain"

func GetBatchTypeStr(object interface{}) any {
	switch object.(type) {
	case *domain.AddressObject:
		return "address objects"
	case *domain.Param:
		return "params"
	case *domain.HouseType:
		return "HOUSETYPE"
	case *domain.AddressObjectType:
		return "ADDRESSOBJECTTYPE"
	case *domain.ApartmentType:
		return "APARTMENTTYPE"
	case *domain.NdocKind:
		return "NDOCKIND"
	case *domain.NdocType:
		return "NDOCTYPE"
	case *domain.ObjectLevel:
		return "OBJECTLEVEL"
	case *domain.OperationType:
		return "OPERATIONTYPE"
	case *domain.ParamType:
		return "PARAMTYPE"
	case *domain.RoomType:
		return "ROOMTYPE"
	case *domain.AdmHierarchyItem:
		return "ADMHIERARCHYITEM"
	default:
		return "unknown type"
	}
}
