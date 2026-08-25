package domain

import "encoding/xml"

type AdmHierarchyItem struct {
	ID          int    `xml:"ID,attr"`
	ObjectID    int    `xml:"OBJECTID,attr"`
	ParentObjID int    `xml:"PARENTOBJID,attr"`
	ChangeID    int    `xml:"CHANGEID,attr"`
	RegionCode  int    `xml:"REGIONCODE,attr"`
	PrevID      int    `xml:"PREVID,attr"`
	NextID      int    `xml:"NEXTID,attr"`
	UpdateDate  string `xml:"UPDATEDATE,attr"`
	StartDate   string `xml:"STARTDATE,attr"`
	EndDate     string `xml:"ENDDATE,attr"`
	IsActive    int    `xml:"ISACTIVE,attr"`
	Path        string `xml:"PATH,attr"`
}

func (o *AdmHierarchyItem) Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error) {
	err := decoder.DecodeElement(o, se)
	if err != nil {
		return nil, err
	}

	return o, nil
}
