package domain

import "encoding/xml"

type RoomType struct {
	ID         int    `xml:"ID,attr"`
	Name       string `xml:"NAME,attr"`
	Desc       string `xml:"DESC,attr"`
	IsActive   bool   `xml:"ISACTIVE,attr"`
	StartDate  string `xml:"STARTDATE,attr"`
	EndDate    string `xml:"ENDDATE,attr"`
	UpdateDate string `xml:"UPDATEDATE,attr"`
}

func (o *RoomType) Decode(decoder *xml.Decoder, se *xml.StartElement) (interface{}, error) {
	err := decoder.DecodeElement(o, se)
	if err != nil {
		return nil, err
	}
	return o, nil
}
