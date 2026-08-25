package domain

import (
	"encoding/xml"
)

type AddressObject struct {
	Id         int    `xml:"ID,attr"`
	ObjectId   string `xml:"OBJECTID,attr"`
	ObjectGuid string `xml:"OBJECTGUID,attr"`
	ChangeId   int    `xml:"CHANGEID,attr"`
	Name       string `xml:"NAME,attr"`
	TypeName   string `xml:"TYPENAME,attr"`
	Level      int    `xml:"LEVEL,attr"`
	OperTypeId int    `xml:"OPERTYPEID,attr"`
	PrevId     int    `xml:"PREVID,attr"`
	NextId     int    `xml:"NEXTID,attr"`
	UpdateDate string `xml:"UPDATEDATE,attr"`
	StartDate  string `xml:"STARTDATE,attr"`
	EndDate    string `xml:"ENDDATE,attr"`
	IsActual   int    `xml:"ISACTUAL,attr"`
	IsActive   int    `xml:"ISACTIVE,attr"`
}

func (o *AddressObject) Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error) {
	err := decoder.DecodeElement(o, se)
	if err != nil {
		return nil, err
	}

	return o, nil
}
